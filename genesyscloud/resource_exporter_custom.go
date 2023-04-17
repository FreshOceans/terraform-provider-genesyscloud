package genesyscloud

import (
	"fmt"
	"log"
	"os"
	"path"
)

/*
The resource_genesyscloud_routing_queue object has the concept of bullseye ring with a member_groups attribute.
The routing team has overloaded the meaning of the member_groups so you can id and then define what "type" of id this is.
This causes problems with the exporter because our export process expects id to map to a specific resource.

This customer custom router will look at the member_group_type and resolve whether it is SKILLGROUP, GROUP type.  It will then
find the appropriate resource out of the exporters and build a reference appropriately.
*/
func MemberGroupsResolver(configMap map[string]interface{}, exporters map[string]*ResourceExporter) error {

	memberGroupType := configMap["member_group_type"]
	memberGroupID := configMap["member_group_id"].(string)

	switch memberGroupType {
	case "SKILLGROUP":
		if exporter, ok := exporters["genesyscloud_routing_skill_group"]; ok {
			exportId := (*exporter.SanitizedResourceMap[memberGroupID]).Name
			configMap["member_group_id"] = fmt.Sprintf("${genesyscloud_routing_skill_group.%s.id}", exportId)
		} else {
			return fmt.Errorf("unable to locate genesyscloud_routing_skill_group in the exporters array. Unable to resolve the ID for the member group resource")
		}

	case "GROUP":
		if exporter, ok := exporters["genesyscloud_group"]; ok {
			exportId := (*exporter.SanitizedResourceMap[memberGroupID]).Name
			configMap["member_group_id"] = fmt.Sprintf("${genesyscloud_group.%s.id}", exportId)
		} else {
			return fmt.Errorf("unable to locate genesyscloud_group in the exporters array. Unable to resolve the ID for the member group resource")
		}
	default:
		return fmt.Errorf("the memberGroupType %s cannot be located. Can not resolve to a reference attribute", memberGroupType)
	}

	return nil
}

func FileContentHashResolver(configMap map[string]interface{}, exporters map[string]*ResourceExporter) error {
	filepath := configMap["filepath"]
	flowId := configMap["id"]
	exporter := exporters["genesyscloud_flow"]
	
	writeToFile("***Writing***", "/Users/dginty/genesys_src/repos/bug-replication/exporter.txt")
	writeToFile(fmt.Sprintf("Config file: %v", configMap), "/Users/dginty/genesys_src/repos/bug-replication/exporter.txt")
	if filepath == nil && flowId != nil {
		flow := flowId.(string)
		writeToFile(fmt.Sprintf("Exporter flow: %v", (*exporter.SanitizedResourceMap[flow]).Name), "/Users/dginty/genesys_src/repos/bug-replication/exporter.txt")
	}
	writeToFile(fmt.Sprintf("Filepath: %v", filepath), "/Users/dginty/genesys_src/repos/bug-replication/exporter.txt")
	writeToFile("***Done***\n", "/Users/dginty/genesys_src/repos/bug-replication/exporter.txt")

	configMap["file_content_hash"] = fmt.Sprintf("filesha256(%s)", filepath)

	return nil
}

func ArchitectPromptAudioResolver(promptId string, exportDirectory string, subDirectory string, configMap map[string]interface{}, meta interface{}) error {
	fullPath := path.Join(exportDirectory, subDirectory)
	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		return err
	}

	audioDataList, err := getArchitectPromptAudioData(promptId, meta)
	if err != nil || len(audioDataList) == 0 {
		return err
	}

	for _, data := range audioDataList {
		if err := downloadAudioFile(fullPath, data.FileName, data.MediaUri); err != nil {
			return err
		}
	}
	updateFilenamesInExportConfigMap(configMap, audioDataList, subDirectory)
	return nil
}

// Function to write to file - Go
func writeToFile(input string, destination string) {
	f, err := os.OpenFile(destination,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	f.WriteString(input + "\n")
}
