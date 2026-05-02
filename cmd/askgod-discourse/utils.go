package main

import "slices"

func int64InSlice(key int64, list []int64) bool {
	return slices.Contains(list, key)
}

func stringInSlice(key string, list []string) bool {
	return slices.Contains(list, key)
}
