package logs

import (
	"log"
	"os"
)

var ErrorLog = log.New(os.Stdout, "error  --> ", log.LstdFlags)
var InfoLog = log.New(os.Stdout, "info --> ", log.LstdFlags)
var DebugLog = log.New(os.Stdout, "debug --> ", log.LstdFlags)
