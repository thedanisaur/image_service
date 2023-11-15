package util

import (
	"math/rand"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func FormatName(title string) string {
	name := regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(title, "")
	name = strings.ReplaceAll(name, " ", "_")
	return strings.ToLower(name)
}

func GetRandomInt(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}
