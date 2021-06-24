package init

import (
    "fmt"
    "os"
    // "os/exec"
	"io/ioutil"

	. "antrea-audit/git-manager/gitops"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	billy "github.com/go-git/go-billy/v5"
	memory "github.com/go-git/go-git/v5/storage/memory"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
    "github.com/go-git/go-git/v5"
)

func SetupRepo(k *Kubernetes, dir *string) error {
	r, err := createRepo(k, dir)
	if err != nil {
		return errors.WithMessagef(err, "couldn't create repo")
	}
	if err := addResources(k, *dir); err != nil {
		return errors.WithMessagef(err, "couldn't write network policies")
	}
	AddAndCommit(r, "audit-init", "system@audit.antrea.io", "Initial commit of existing policies")
	fmt.Println("Repository successfully initialized")
	return nil
}

func createRepo(k *Kubernetes, dir *string) (*git.Repository, error) {
    if *dir == "" {
        path, err := os.Getwd()
        if err != nil {
            return nil, errors.WithMessagef(err, "could not retrieve the current working directory")
        }
        *dir = path
    }
    //cmd := exec.Command("sudo", "chmod", "a+w", "./")
	//err = cmd.Run()
    // if err != nil {
        //return nil, err
    //}
    *dir += "/network-policy-repository"
    r, err := git.PlainInit(*dir, false)
    if err == git.ErrRepositoryAlreadyExists {
		r, err = git.PlainOpen(*dir)
        if err != nil {
            return nil, errors.WithMessagef(err, "repo exists and cannot open")
        }
        return r, nil
	} else if err != nil {
		return nil, errors.WithMessagef(err, "could not initialize git repo")
	}
	return r, nil
}

func SetupRepoInMem(k *Kubernetes, storer *memory.Storage, fs billy.Filesystem) error {
	r, err := git.Init(storer, fs)
    if err == git.ErrRepositoryAlreadyExists {
		return nil 
	} else if err != nil {
		return errors.WithMessagef(err, "could not initialize git repo")
	}
	if err := addResourcesInMem(k, fs); err != nil {
		return errors.WithMessagef(err, "couldn't write network policies")
	}
	AddAndCommit(r, "audit-init", "system@audit.antrea.io", "initial commit of existing policies")
	fmt.Println("Repository successfully initialized")
	return nil
}

func addResources(k *Kubernetes, dir string) error {
    os.Mkdir(dir + "/k8s-policies", 0700)
    os.Mkdir(dir + "/antrea-policies", 0700)
    os.Mkdir(dir + "/antrea-cluster-policies", 0700)
	os.Mkdir(dir + "/antrea-tiers", 0700)
	if err := addK8sPolicies(k, dir); err != nil {
		return err
	}
	if err := addAntreaPolicies(k, dir); err != nil {
		return err
	}
	if err := addAntreaClusterPolicies(k, dir); err != nil {
		return err
	}
	if err := addAntreaTiers(k, dir); err != nil {
		return err
	}
	return nil
}

func addResourcesInMem(k *Kubernetes, fs billy.Filesystem) error {
	fs.MkdirAll("k8s-policies", 0700)
	fs.MkdirAll("antrea-policies", 0700)
	fs.MkdirAll("antrea-cluster-policies", 0700)
	fs.MkdirAll("antrea-tiers", 0700)
	if err := addK8sPoliciesInMem(k, fs); err != nil {
		return err
	}
	if err := addAntreaPoliciesInMem(k, fs); err != nil {
		return err
	}
	if err := addAntreaClusterPoliciesInMem(k, fs); err != nil {
		return err
	}
	if err := addAntreaTiersInMem(k, fs); err != nil {
		return err
	}
	return nil
}

func addK8sPolicies(k *Kubernetes, dir string) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(dir + "/k8s-policies/" + np.Namespace, 0700)
		}
		path := dir + "/k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addK8sPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetK8sPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			fs.MkdirAll("k8s-policies/" + np.Namespace, 0700)
		}
		path := "k8s-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaPolicies(k *Kubernetes, dir string) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			os.Mkdir(dir + "/antrea-policies/" + np.Namespace, 0700)
		}
		path := dir + "/antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addAntreaPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetAntreaPolicies()
	if err != nil {
		return err
	}
	var namespaces []string
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "NetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		if !stringInSlice(np.Namespace, namespaces) {
			namespaces = append(namespaces, np.Namespace)
			fs.MkdirAll("antrea-policies/" + np.Namespace, 0700)
		}
		path := "antrea-policies/" + np.Namespace + "/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaClusterPolicies(k *Kubernetes, dir string) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := dir + "/antrea-cluster-policies/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write policy config to file")
		}
	}
	return nil
}

func addAntreaClusterPoliciesInMem(k *Kubernetes, fs billy.Filesystem) error {
	policies, err := k.GetAntreaClusterPolicies()
	if err != nil {
		return err
	}
	for _, np := range policies.Items {
		np.TypeMeta = metav1.TypeMeta{
			Kind: "ClusterNetworkPolicy",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := "antrea-cluster-policies/" + np.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&np)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal policy config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func addAntreaTiers(k *Kubernetes, dir string) error {
	tiers, err := k.GetAntreaTiers()
	if err != nil {
		return err
	}
	for _, tier := range tiers.Items {
		tier.TypeMeta = metav1.TypeMeta{
			Kind: "Tier",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := dir + "/antrea-tiers/" + tier.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&tier)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal tier config")
		}
		err = ioutil.WriteFile(path, y, 0644)
		if err != nil {
			return errors.Wrapf(err, "unable to write tier config to file")
		}
	}
	return nil
}

func addAntreaTiersInMem(k *Kubernetes, fs billy.Filesystem) error {
	tiers, err := k.GetAntreaTiers()
	if err != nil {
		return err
	}
	for _, tier := range tiers.Items {
		tier.TypeMeta = metav1.TypeMeta{
			Kind: "Tier",
			APIVersion: "crd.antrea.io/v1alpha1",
		}
		path := "antrea-tiers/" + tier.Name + ".yaml"
		fmt.Println("Added "+path)
		y, err := yaml.Marshal(&tier)
		if err != nil {
			return errors.Wrapf(err, "unable to marshal tier config")
		}
		newFile, err := fs.Create(path)
		if err != nil {
			return errors.Wrapf(err, "unable to create file")
		}
		newFile.Write(y)
		newFile.Close()
	}
	return nil
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}
