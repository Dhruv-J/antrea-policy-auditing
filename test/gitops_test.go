package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"antrea-audit/gitops"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	directory = ""
	np1 = &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
		},
	}
	np2 = &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
	anp1 = &crdv1alpha1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "crd.antrea.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "anpA", UID: "uidC"},
		Spec: crdv1alpha1.NetworkPolicySpec{
			AppliedTo: []crdv1alpha1.NetworkPolicyPeer{
				{PodSelector: &selectorA},
			},
			Priority: p10,
			Ingress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int80,
						},
					},
					From: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
			Egress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int81,
						},
					},
					To: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
		},
	}
)

func TestHandleEventList(t *testing.T) {
	fakeClient := NewClient(np1.DeepCopy(), anp1.DeepCopy())
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}

	jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
	if err != nil {
		fmt.Println(err)
		t.Errorf("could not read audit-log file")
	}

	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, directory)
	if err != nil {
		fmt.Println(err)
		t.Errorf("could not set up repo")
	}

	err = cr.HandleEventList(jsonStr)
	if err != nil {
		fmt.Println(err)
		t.Errorf("could not handle audit event list")
	}
}

func TestTagging(t *testing.T) {
	fakeClient := NewClient()
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}
	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, directory)
	if err != nil {
		t.Errorf("Error (TestTagging): unable to set up repo")
	}
	h, err := cr.Repo.Head()
	if err != nil {
		t.Errorf("Error (TestTagging): unable to get repo head ref")
	}

	testSig := &object.Signature{
		Name:  "test",
		Email: "test@antrea.audit.io",
		When:  time.Now(),
	}
	// Attempt to add tag to nonexistent commit
	if err := cr.TagCommit("bad-hash", "test-tag", testSig); err == nil {
		t.Errorf("Error (TestTagging): should have returned error on bad commit hash")
	}

	// Create new tags successfully
	if err := cr.TagCommit(h.Hash().String(), "test-tag", testSig); err != nil {
		t.Errorf("Error (TestTagging): unable to create new tag")
	}
	if err := cr.TagCommit(h.Hash().String(), "test-tag-2", testSig); err != nil {
		t.Errorf("Error (TestTagging): unable to create new tag")
	}
	_, err = cr.Repo.Tag("test-tag")
	if err != nil {
		t.Errorf("Error (TestTagging): could not retrieve created tag")
	}
	_, err = cr.Repo.Tag("test-tag-2")
	if err != nil {
		t.Errorf("Error (TestTagging): could not retrieve created tag")
	}

	// Attempt to add tag with the same name
	if err := cr.TagCommit(h.Hash().String(), "test-tag", testSig); err != nil {
		t.Errorf("Error (TestTagging): unable to handle duplicate tag creation")
	}
	tags, _ := cr.Repo.TagObjects()
	tagCount := 0
	if err := tags.ForEach(func(tag *object.Tag) error {
		tagCount += 1
		return nil
	}); err != nil {
		t.Errorf("Error (TestTagging): could not iterate through repo tags")
	}
	assert.Equal(t, 2, tagCount, "Error (TestTagging): duplicate tag detected, tag count should have been 2")
}

func TestRollback(t *testing.T) {
	fakeClient := NewClient(np1.DeepCopy(), anp1.DeepCopy())
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}
	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, directory)
	if err != nil {
		t.Errorf("Error (TestRollback): unable to set up repo")
	}
	h, err := cr.Repo.Head()
	if err != nil {
		t.Errorf("Error (TestRollback): unable to get repo head ref")
	}
	testSig := &object.Signature{
		Name:  "test",
		Email: "test@antrea.audit.io",
		When:  time.Now(),
	}
	if err := cr.TagCommit(h.Hash().String(), "test-tag", testSig); err != nil {
		t.Errorf("Error (TestRollback): unable to create new tag")
	}

	// Create, update, and delete a resource
	r := unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	j, err := json.Marshal(np2)
	if err != nil {
		t.Errorf("Error (TestRollback): unable to convert to json")
	}
	if err := json.Unmarshal(j, &r); err != nil {
		t.Errorf("Error (TestRollback): unable to unmarshal into unstructured object")
	}
	if err := k8s.CreateOrUpdateResource(&r); err != nil {
		t.Errorf("Error (TestRollback): unable to create new resource")
	}

	updatedNP := np1
	updatedNP.ObjectMeta.SetClusterName("new-cluster-name")
	r = unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	j, err = json.Marshal(updatedNP)
	if err != nil {
		t.Errorf("Error (TestRollback): unable to convert to json")
	}
	fmt.Println(string(j))
	if err := json.Unmarshal(j, &r); err != nil {
		t.Errorf("Error (TestRollback): unable to unmarshal into unstructured object")
	}
	if err := k8s.CreateOrUpdateResource(&r); err != nil {
		t.Errorf("Error (TestRollback): unable to update resource")
	}
	fmt.Println("got here")

	r = unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy",
	})
	j, err = json.Marshal(anp1)
	if err != nil {
		t.Errorf("Error (TestRollback): unable to convert to json")
	}
	if err := json.Unmarshal(j, &r); err != nil {
		t.Errorf("Error (TestRollback): unable to unmarshal into unstructured object")
	}
	if err := k8s.DeleteResource(&r); err != nil {
		t.Errorf("Error (TestRollback): unable to delete resource")
	}

	jsonStr, err := ioutil.ReadFile("./files/rollback-log.txt")
	if err != nil {
		t.Errorf("could not read rollback-log file")
	}
	if err := cr.HandleEventList(jsonStr); err != nil {
		t.Errorf("could not process audit events from file")
	}

	// Attempt rollback
	commit, err := cr.TagToCommit("test-tag")
	if err != nil {
		t.Errorf("Error (TestRollback): could not retrieve commit from tag")
	}
	if err := cr.RollbackRepo(commit); err != nil {
		t.Errorf("Error (TestRollback): rollback failed")
	}

	// Check latest commit
	newH, err := cr.Repo.Head()
	if err != nil {
		t.Errorf("Error (TestRollback): unable to get repo head ref")
	}
	rollbackCommit, err := cr.Repo.CommitObject(newH.Hash())
	if err != nil {
		t.Errorf("Error (TestRollback): unable to get rollback commit object")
	}
	assert.Equal(t, "Rollback to commit " + h.Hash().String(), rollbackCommit.Message,
		"Error (TestRollback): rollback commit not found, head commit message mismatch")

	// Check cluster state
	res := &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	np, err := k8s.GetResource(res, "nsA", "npA")
	if err != nil {
		t.Errorf("Error (TestRollback): unable to get policy after rollback")
	}
	assert.Equal(t, "", np.GetClusterName(), 
		"Error (TestRollback): updated field should be empty after rollback")

	res = &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy",
	})
	_, err = k8s.GetResource(res, "nsA", "anpA")
	if err != nil {
		t.Errorf("Error (TestRollback): unable to get antrea policy after rollback")
	}
}
