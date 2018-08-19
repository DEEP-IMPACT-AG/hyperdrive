// ## Release
//
// Releasing is more involved that building. First, we release the common
// part of the core to all regions known by the hyperdrive and then we
// release the cloudfront rewrite lambda@edge to the us-east-1 regions.
package release

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/DEEP-IMPACT-AG/hyperdrive/tools/assemble/build"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

func Release() error {
	version, err := HyperdriveVersion()
	if err != nil {
		return err
	}
	log.Println("Deploying version", version)
	err = zipFunctions()
	if err != nil {
		return err
	}
	regions, err := hyperdriveRegions()
	if err != nil {
		return err
	}
	for _, region := range regions {
		log.Println("Deploying region", region)
		err := releaseRegion(region, version, false)
		if err != nil {
			return errors.Wrapf(err, "could not release region %s", region)
		}
	}
	return releaseCfRewrite(version, false)
}

func zipFunctions() error {
	if err := os.RemoveAll("dist"); err != nil {
		return err
	}
	if err := os.Mkdir("dist", 0700); err != nil {
		return err
	}
	for _, function := range build.Functions {
		dir, err := build.FunctionDir(function)
		if err != nil {
			return err
		}
		if err := zipFunction(dir); err != nil {
			return err
		}
	}
	return nil
}

func zipFunction(dir string) error {
	functionName := filepath.Base(dir)
	zipFile := fmt.Sprintf("../make/dist/%s.zip", functionName)
	cmd := exec.Command("zip", "-9", zipFile, functionName)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "could not zip the lambda function %s", functionName)
	}
	return nil
}

//
// The regions supported by the hyperdrive are given by all the regions
// that participate in the stack set `hyperdriveS3Buckets` since the
// hyperdrive S3 buckets have been created via this stack set. The stackset
// is found in the eu-west-1 region.
//
func hyperdriveRegions() ([]string, error) {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not load aws configuration")
	}
	cfs := cloudformation.New(cfg)
	stackSetName := "hyperdriveS3Buckets"
	res, err := cfs.ListStackInstancesRequest(&cloudformation.ListStackInstancesInput{
		StackSetName: &stackSetName,
	}).Send()
	if err != nil {
		return nil, errors.Wrapf(err, "could not fetch the stack instances of stack set %s", stackSetName)
	}
	regions := make([]string, len(res.Summaries))
	for i, sum := range res.Summaries {
		regions[i] = *sum.Region
	}
	return regions, nil
}

//
// The version being built can be found via `git describe`
//
func HyperdriveVersion() (string, error) {
	// TODO@stan: find a way to call `git describe` directly.
	buf := strings.Builder{}
	path, err := filepath.Abs("./version.sh")
	if err != nil {
		return "", errors.Wrap(err, "script version.sh not found")
	}
	cmd := &exec.Cmd{
		Path: path,
		Args: []string{"version.sh"},
	}
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "could not run the script version.sh")
	}
	return strings.TrimSpace(buf.String()), nil
}

//
// To release for one specific region, we need first to instantiate a S3
// client with the proper region using the standard credential chain of the
// AWS SDK. Then we can upload all the functions, create the hyperdrive
// template and updload it as well.
//
func releaseRegion(region string, hyperdriveVersion string, isTest bool) error {
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithRegion(region),
	)
	if err != nil {
		return errors.Wrapf(err, "could not get aws config for region %s", region)
	}
	s3s := s3.New(cfg)
	var bucket string
	if isTest {
		bucket = region + ".test.hyperdrive.sh"
	} else {
		bucket = region + ".hyperdrive.sh"
	}
	// 1. First, We upload the zipped functions and gather the new versions
	versions := make(map[string]string, len(build.Functions)+1)
	versions["bucket"] = bucket
	for _, function := range build.Functions {
		log.Println("..Uploading function", function)
		versions, err = uploadFunction(s3s, bucket, function, versions)
		if err != nil {
			return err
		}
	}
	// 2. Then, we render the hyperdrive template.
	t, err := template.New("hyperdrive.yaml").ParseFiles("release/hyperdrive.yaml")
	if err != nil {
		return errors.Wrap(err, "could not parse the template hyperdrive.yaml")
	}
	tmpFile, err := ioutil.TempFile("", "hyperdrive")
	if err != nil {
		return errors.Wrap(err, "could not create temporary file with prefix hyperdrive")
	}
	defer tmpFile.Close()
	err = t.Execute(tmpFile, versions)
	if err != nil {
		return errors.Wrapf(err, "could not execute the template hyperdrive.yaml in the temporary file %s", tmpFile.Name())
	}
	log.Println("..Uploading the hyperdriveCore.yaml cloudformation template.")
	// 3. Finally, we upload the rendered template.
	key := "cf/hyperdrive/" + hyperdriveVersion + "/hyperdriveCore.yaml"
	_, err = s3s.PutObjectRequest(&s3.PutObjectInput{
		ACL:    s3.ObjectCannedACLPublicRead,
		Body:   tmpFile,
		Bucket: &bucket,
		Key:    &key,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not upload the file %s with key %s in the bucket %s", tmpFile.Name(), key, bucket)
	}
	return nil
}

func uploadFunction(s3s *s3.S3, bucket, function string, versions map[string]string) (map[string]string, error) {
	zipFileName := function + ".zip"

	zipFile, err := os.Open("dist/" + zipFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open zip file of function %s", function)
	}
	defer zipFile.Close()

	key := "lambda/" + zipFileName
	res, err := s3s.PutObjectRequest(&s3.PutObjectInput{
		ACL:    s3.ObjectCannedACLPublicRead,
		Body:   zipFile,
		Bucket: &bucket,
		Key:    &key,
	}).Send()
	if err != nil {
		return nil, errors.Wrapf(err,"could not upload the function code %s with key %s in the bucket %s", function, key, bucket)
	}
	versions[function] = *res.VersionId
	return versions, nil
}

//
// Releasing the cloudfront folder rewrite lambda@edge function is similar
// to the hyperdrive template: the lambda function is written using node as
// required by lambda@edge and is directly embedded into the template.
//
// There is a trick with the log groups for the lambda@edge function: since
// it logs in every region (it always choose the closest region to log to),
// we create log groups for every existing region in order to set their
// retention period to 90 days.
//
type CFRewriteLogGroup struct {
	Region, Infix string
}

func releaseCfRewrite(hyperdriveVersion string, isTest bool) error {
	log.Println("Create the CF Rewrite Template in the us-east-1 region")
	// 1. we release only in the us-east-region as Lambdas for CF must be in the us-east-1 region.
	cfg, err := external.LoadDefaultAWSConfig(
		external.WithRegion("us-east-1"),
	)
	if err != nil {
		return errors.Wrap(err, "could not get aws config for region us-east-1")
	}
	s3s := s3.New(cfg)
	var bucket string
	if isTest {
		bucket = "us-east-1.test.hyperdrive.sh"
	} else {
		bucket = "us-east-1.hyperdrive.sh"
	}
	// 2. We load a skeleton for the cloudfront folder rewrite
	box := packr.NewBox(".")
	var cfRewrite map[interface{}]interface{}
	yml, err := box.MustBytes("cf_rewrite.yaml")
	if err != nil {
		return errors.Wrap(err, "could not load the file cf_rewrite.yaml")
	}
	if err := yaml.Unmarshal(yml, &cfRewrite); err != nil {
		return errors.Wrap(err, "could not unmarshal the file cf_rewrite.yaml")
	}
	cfLogGroup, err := template.
		New("cf_rewrite_log_group.yaml").
		ParseFiles("release/cf_rewrite_log_group.yaml")
	if err != nil {
		return errors.Wrap(err, "could not parse the template cf_rewrite_log_group.yaml")
	}
	// 3. We fetch all the existing regions from EC2
	ec2s := ec2.New(cfg)
	res, err := ec2s.DescribeRegionsRequest(&ec2.DescribeRegionsInput{}).Send()
	if err != nil {
		return errors.Wrap(err, "could not fetch all ec2 regions")
	}
	// 4. We create the log group resources and add them to the template.
	resources := cfRewrite["Resources"].(map[interface{}]interface{})
	for _, region := range res.Regions {
		var infix string
		regionName := *region.RegionName
		if regionName != "us-east-1" {
			infix = "us-east-1."
		}
		buf := new(bytes.Buffer)
		if err := cfLogGroup.Execute(buf, CFRewriteLogGroup{
			Region: regionName,
			Infix:  infix,
		}); err != nil {
			return errors.Wrapf(err, "could not execute template cf_rewrite_log_group.yaml for the region %s", region)
		}
		logGroup := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(buf.Bytes(), &logGroup); err != nil {
			return errors.Wrapf(err, "could not unmarshal for the region %s", region)
		}
		resources[cfRewriteLogGroupName(regionName)] = logGroup
	}
	// 5. We write the template to a temporary file.
	tmpBytes, err := yaml.Marshal(cfRewrite)
	if err != nil {
		return errors.Wrap(err, "could not marshal the full template")
	}
	tmpFile, err := ioutil.TempFile("", "cfrewrite")
	if err != nil {
		return errors.Wrap(err, "could not create temporary file with prefix cfrewrite")
	}
	defer tmpFile.Close()
	writer := bufio.NewWriter(tmpFile)
	if _, err = writer.Write(tmpBytes); err != nil {
		return errors.Wrapf(err, "could not write to the file %s", tmpFile.Name())
	}
	if err = writer.Flush(); err != nil {
		return errors.Wrapf(err, "could not flush the file %s", tmpFile.Name())
	}
	// 6. Finally, we upload the template.
	log.Println("..Uploading the cfRewrite.yaml cloudformation template.")
	key := "cf/hyperdrive/" + hyperdriveVersion + "/cfRewrite.yaml"
	_, err = s3s.PutObjectRequest(&s3.PutObjectInput{
		ACL:    s3.ObjectCannedACLPublicRead,
		Body:   tmpFile,
		Bucket: &bucket,
		Key:    &key,
	}).Send()
	if err != nil {
		return errors.Wrapf(err, "could not upload the file %s to the key %s in the bucket %", tmpFile.Name(), key, bucket)
	}
	return nil
}

func cfRewriteLogGroupName(region string) string {
	b := strings.Builder{}
	b.WriteString("CfRewriteLogGroup")
	for _, el := range strings.Split(region, "-") {
		b.WriteString(strings.Title(el))
	}
	return b.String()
}
