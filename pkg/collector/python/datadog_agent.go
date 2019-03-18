// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build python

package python

import (
	"encoding/json"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/kubernetes/clustername"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/version"
)

/*
#cgo LDFLAGS: -ldatadog-agent-six -ldl
#include "datadog_agent_six.h"
*/
import (
	"C"
)

// GetVersion exposes the version of the agent to Python checks.
//export GetVersion
func GetVersion(agentVersion **C.char) {
	av, _ := version.New(version.AgentVersion, version.Commit)
	// version will be free by six when it's done with it
	*agentVersion = C.CString(av.GetNumber())
}

// GetHostname exposes the current hostname of the agent to Python checks.
//export GetHostname
func GetHostname(hostname **C.char) {
	goHostname, err := util.GetHostname()
	if err != nil {
		log.Warnf("Error getting hostname: %s\n", err)
		goHostname = ""
	}
	// hostname will be free by six when it's done with it
	*hostname = C.CString(goHostname)
}

// GetClusterName exposes the current clustername (if it exists) of the agent to Python checks.
//export GetClusterName
func GetClusterName(clusterName **C.char) {
	goClusterName := clustername.GetClusterName()
	// clusterName will be free by six when it's done with it
	*clusterName = C.CString(goClusterName)
}

// Headers returns a basic set of HTTP headers that can be used by clients in Python checks.
//export Headers
func Headers(jsonPayload **C.char) {
	h := util.HTTPHeaders()

	data, err := json.Marshal(h)
	if err != nil {
		log.Errorf("datadog_agent: could not Marshal headers: %s", err)
		*jsonPayload = nil
		return
	}
	// jsonPayload will be free by six when it's done with it
	*jsonPayload = C.CString(string(data))
}

// GetConfig returns a value from the agent configuration.
// Indirectly used by the C function `get_config` that's mapped to `datadog_agent.get_config`.
//export GetConfig
func GetConfig(key *C.char, jsonPayload **C.char) {
	goKey := C.GoString(key)
	if !config.Datadog.IsSet(goKey) {
		*jsonPayload = nil
		return
	}

	value := config.Datadog.Get(goKey)
	data, err := json.Marshal(value)
	if err != nil {
		log.Errorf("datadog_agent: could not convert configuration value (%v) to python types: %s", value, err)
		*jsonPayload = nil
		return
	}
	// jsonPayload will be free by six when it's done with it
	*jsonPayload = C.CString(string(data))
}

// LogMessage logs a message from python through the agent logger (see
// https://docs.python.org/2.7/library/logging.html#logging-levels)
//export LogMessage
func LogMessage(message *C.char, logLevel C.int) {
	goMsg := C.GoString(message)

	switch logLevel {
	case 50: // CRITICAL
		log.Critical(goMsg)
	case 40: // ERROR
		log.Error(goMsg)
	case 30: // WARNING
		log.Warn(goMsg)
	case 20: // INFO
		log.Info(goMsg)
	case 10: // DEBUG
		log.Debug(goMsg)
	// Custom log level defined in:
	// https://github.com/DataDog/integrations-core/blob/master/datadog_checks_base/datadog_checks/base/log.py
	case 7: // TRACE
		log.Trace(goMsg)
	default: // unknown log level
		log.Info(goMsg)
	}

	return
}

//// SetExternalTags adds a set of tags for a given hostnane to the External Host
//// Tags metadata provider cache.
//// Indirectly used by the C function `set_external_tags` that's mapped to `datadog_agent.set_external_tags`.
////export SetExternalTags
//func SetExternalTags(tags *C.char) {
//	hname := C.GoString(hostname)
//	stype := C.GoString(sourceType)
//	tlen := int(tagsLen)
//
//	// The maximum capacity of the following slice is limited to (2^29)-1 to remain compatible
//	// with 32-bit platforms. The size of a `*C.char` (a pointer) is 4 Byte on a 32-bit system
//	// and (2^29)*4 == math.MaxInt32 + 1. -- See issue golang/go#13656
//	tagsSlice := (*[1<<29 - 1]*C.char)(unsafe.Pointer(tags))[:tlen:tlen]
//	tagsStrings := []string{}
//
//	for i := 0; i < tlen; i++ {
//		tag := C.GoString(tagsSlice[i])
//		tagsStrings = append(tagsStrings, tag)
//	}
//
//	externalhost.SetExternalTags(hname, stype, tagsStrings)
//	return C._none()
//}
//
//// GetSubprocessOutput runs the subprocess and returns the output
//// Indirectly used by the C function `get_subprocess_output` that's mapped to `_util.get_subprocess_output`.
////export GetSubprocessOutput
//func GetSubprocessOutput(argv **C.char, argc, raise int) *C.PyObject {
//
//	// IMPORTANT: this is (probably) running in a go routine already locked to
//	//            a thread. No need to do it again, and definitely no need to
//	//            to release it - we can let the caller do that.
//
//	// https://github.com/golang/go/wiki/cgo#turning-c-arrays-into-go-slices
//
//	threadState := SaveThreadState()
//
//	length := int(argc)
//	subprocessArgs := make([]string, length-1)
//
//	// The maximum capacity of the following slice is limited to (2^29)-1 to remain compatible
//	// with 32-bit platforms. The size of a `*C.char` (a pointer) is 4 Byte on a 32-bit system
//	// and (2^29)*4 == math.MaxInt32 + 1. -- See issue golang/go#13656
//	cmdSlice := (*[1<<29 - 1]*C.char)(unsafe.Pointer(argv))[:length:length]
//
//	subprocessCmd := C.GoString(cmdSlice[0])
//	for i := 1; i < length; i++ {
//		subprocessArgs[i-1] = C.GoString(cmdSlice[i])
//	}
//	ctx, _ := GetSubprocessContextCancel()
//	cmd := exec.CommandContext(ctx, subprocessCmd, subprocessArgs...)
//
//	stdout, err := cmd.StdoutPipe()
//	if err != nil {
//		glock := RestoreThreadStateAndLock(threadState)
//		defer C.PyGILState_Release(glock)
//
//		cErr := C.CString(fmt.Sprintf("internal error creating stdout pipe: %v", err))
//		C.PyErr_SetString(C.PyExc_Exception, cErr)
//		C.free(unsafe.Pointer(cErr))
//		return nil
//	}
//
//	var wg sync.WaitGroup
//	var output []byte
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//		output, _ = ioutil.ReadAll(stdout)
//	}()
//
//	stderr, err := cmd.StderrPipe()
//	if err != nil {
//		glock := RestoreThreadStateAndLock(threadState)
//		defer C.PyGILState_Release(glock)
//
//		cErr := C.CString(fmt.Sprintf("internal error creating stderr pipe: %v", err))
//		C.PyErr_SetString(C.PyExc_Exception, cErr)
//		C.free(unsafe.Pointer(cErr))
//		return nil
//	}
//
//	var outputErr []byte
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//		outputErr, _ = ioutil.ReadAll(stderr)
//	}()
//
//	cmd.Start()
//
//	// Wait for the pipes to be closed *before* waiting for the cmd to exit, as per os.exec docs
//	wg.Wait()
//
//	retCode := 0
//	err = cmd.Wait()
//	if exiterr, ok := err.(*exec.ExitError); ok {
//		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
//			retCode = status.ExitStatus()
//		}
//	}
//
//	glock := RestoreThreadStateAndLock(threadState)
//	defer C.PyGILState_Release(glock)
//
//	if raise > 0 {
//		// raise on error
//		if len(output) == 0 {
//			cModuleName := C.CString("_util")
//			utilModule := C.PyImport_ImportModule(cModuleName)
//			C.free(unsafe.Pointer(cModuleName))
//			if utilModule == nil {
//				return nil
//			}
//			defer C.Py_DecRef(utilModule)
//
//			cExcName := C.CString("SubprocessOutputEmptyError")
//			excClass := C.PyObject_GetAttrString(utilModule, cExcName)
//			C.free(unsafe.Pointer(cExcName))
//			if excClass == nil {
//				return nil
//			}
//			defer C.Py_DecRef(excClass)
//
//			cErr := C.CString("get_subprocess_output expected output but had none.")
//			C.PyErr_SetString((*C.PyObject)(unsafe.Pointer(excClass)), cErr)
//			C.free(unsafe.Pointer(cErr))
//			return nil
//		}
//	}
//
//	cOutput := C.CString(string(output[:]))
//	pyOutput := C.PyString_FromString(cOutput)
//	C.free(unsafe.Pointer(cOutput))
//	cOutputErr := C.CString(string(outputErr[:]))
//	pyOutputErr := C.PyString_FromString(cOutputErr)
//	C.free(unsafe.Pointer(cOutputErr))
//	pyRetCode := C.PyInt_FromLong(C.long(retCode))
//
//	pyResult := C.PyTuple_New(3)
//	C.PyTuple_SetItem(pyResult, 0, pyOutput)
//	C.PyTuple_SetItem(pyResult, 1, pyOutputErr)
//	C.PyTuple_SetItem(pyResult, 2, pyRetCode)
//
//	return pyResult
//}