package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"strings"
	"time"
)

func jsonToMap(sJson string) map[string]interface{} {
	var result map[string]interface{}
	if len(sJson) == 0 {
		return result
	}
	json.Unmarshal([]byte(sJson), &result)
	return result
}

func byteToSlice(bytes []uint8) []string {
	if len(bytes) == 0 {
		return []string{}
	}
	sValue := string(bytes)
	result := strings.Split(sValue, "\n")
	return result
}

func byteToMaps(bytes []byte) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)
	if len(bytes) == 0 {
		return result
	}
	mSlice := byteToSlice(bytes)
	if len(mSlice) == 0 {
		return result
	}
	for _, element := range mSlice {
		e := strings.TrimSpace(element)
		if len(e) == 0 {
			continue
		}
		mapEntry := jsonToMap(element)
		result = append(result, mapEntry)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func checkAndReplNil(s interface{}, r string) string {
	if s == nil {
		return r
	}
	res, ok := s.(string)
	if !ok {
		return r
	}
	return res
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func Convert(e []map[string]interface{}) []SoftwareMigration {
	res := make([]SoftwareMigration, 0)
	for _, v := range e {
		sCreationTime := v["creationTime"].(string)
		dateCreationTime, _ := time.Parse(time.RFC3339, sCreationTime)

		res = append(res, SoftwareMigration{
			Software: Software{
				Name:         checkAndReplNil(v["name"], ""),
				Version:      checkAndReplNil(v["version"], ""),
				Url:          checkAndReplNil(v["url"], ""),
				Id:           checkAndReplNil(v["id"], ""),
				CreationTime: dateCreationTime,
			},
			IsOldPackage: isOldSoftwarePackage(v),
			Migrated:     false,
		})
	}
	return res
}

func Filter(ss []SoftwareMigration, test func(SoftwareMigration) bool) (meetCriteria []SoftwareMigration, notMeetCriteria []SoftwareMigration) {
	for _, s := range ss {
		if test(s) {
			meetCriteria = append(meetCriteria, s)
		} else {
			notMeetCriteria = append(notMeetCriteria, s)
		}
	}
	return
}

func GroupByName(ss []SoftwareMigration) map[string][]SoftwareMigration {
	res := make(map[string][]SoftwareMigration)
	for _, s := range ss {
		slice, ok := res[s.Software.Name]
		if !ok {
			slice = make([]SoftwareMigration, 0)
		}
		slice = append(slice, s)
		res[s.Software.Name] = slice
	}
	return res
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
