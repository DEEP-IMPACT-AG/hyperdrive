package main

import "encoding/json"

func dummyResource() map[string]interface{} {
	var result map[string]interface{}
	dum := Resources.Bytes("dummy-resource.json")
	json.Unmarshal(dum, &result)
	return result
}

func accOutput(m map[string]interface{}, key, val string) {
	m[key] = map[string]interface{}{
		"Value": val,
		"Export": map[string]interface{}{
			"Name": map[string]interface{}{
				"Fn::Sub": "${AWS::StackName}-" + key,
			},
		},
	}
}
