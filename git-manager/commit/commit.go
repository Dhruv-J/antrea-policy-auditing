package main

import (
    "fmt"
    "os"
    "time"
    "io/ioutil"
    "encoding/json"

    "github.com/go-git/go-git/v5"
    . "github.com/go-git/go-git/v5/_examples"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/ghodss/yaml"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

var directory string
var dirMap map[string]string{
    "networking.k8s.io/v1NetworkPolicy": "k8s-policy",
    "crd.antrea.io/v1alpha1NetworkPolicy": "antrea-policy",
    "crd.antrea.io/v1alpha1ClusterNetworkPolicy": "antrea-cluster-policy",
}

type yamlMask struct {
    Kind        string `json:"kind"`
    ApiVersion  string `json:"apiVersion"`
}

func eventToCommit(event auditv1.Event) {
    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    CheckIfError(err)
    w, err := r.Worktree()
    CheckIfError(err)

    Info("git add .")
    _, err = w.Add(".")
    CheckIfError(err)

    Info("git commit -m \"Network Policy change commit\"")
    _, err = w.Commit("Network Policy change commit", &git.CommitOptions{
        Author: &object.Signature{
            Name:  event.User.Username,
            Email: event.User.Username+event.User.UID+"@audit.antrea.io",
            When:  time.Now(),
        },
    })
    CheckIfError(err)
}
func EventListToCommit(eventList auditv1.EventList) {
    for _,event := range eventList.Items {
        err := eventToCommit(event)
        if err != nil {
            return err
        }
    }
    return nil
}

func modifyFile(event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.RequestObject.Raw)
    if err!=nil {
        fmt.Printf("error converting json to yaml\n")
        return err
    }

    yMask := yamlMask{}
    err = json.Unmarshal(event.RequestObject.Raw, &yMask)
    if err!=nil {
        fmt.Printf("error unmarshalling json\n")
        return err
    }

    path := directory+"/network-policy-repository/"+dirMap[yMask.ApiVersion+yMask.Kind]+"/"+event.ObjectRef.Namespace
    if _, err := os.Stat(path); os.IsNotExist(err) {
        os.Mkdir(path, 0700)
    }

    path += "/network-policy.yaml"
    err = ioutil.WriteFile(path, y, 0644)
    if err!=nil {
        fmt.Printf("error writing file\n")
        return err
    }

    return nil
}

func ModifyFiles(eventList auditv1.EventList) (error) {
    for _,event := range eventList.Items {
        err := modifyFile(event)
        if err != nil {
            return err
        }
    }
    return nil
}
