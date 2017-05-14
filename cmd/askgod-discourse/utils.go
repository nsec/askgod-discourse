package main

func int64InSlice(key int64, list []int64) bool {
	for _, entry := range list {
		if entry == key {
			return true
		}
	}
	return false
}
