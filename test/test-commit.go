package main

import (
    "fmt"
    "testing"
    "io/ioutil"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func TestEventListToCommit(t *testing.T) {
    jsonStr, _ := ioutil.ReadFile("/files/audit-log.txt")
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonStr, &eventList)

    err = EventListToCommit(eventList)

    if err != nil {
        t.Errorf("Error (TestEventListToCommit): should not return error for correct event list")
    }
}

func TestModifyFiles(t *testing.T) {
    jsonStr, _ := ioutil.ReadFile("/files/audit-log.txt")
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonStr, &eventList)

    err = ModifyFiles(eventList)

    if err != nil {
        t.Errorf("Error (ModifyFiles): should not return error for correct event list")
    }
}

