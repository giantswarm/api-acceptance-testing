// Package kubeconfig contains abilities to create a simple, self-contained kubeconfig YAML file.
package kubeconfig

import (
	"encoding/base64"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type Kubeconfig struct {
	APIVersion     string         `yaml:"apiVersion"`
	Kind           string         `yaml:"kind"`
	Clusters       []NamedCluster `yaml:"clusters"`
	Users          []NamedUser    `yaml:"users"`
	Contexts       []NamedContext `yaml:"contexts"`
	CurrentContext string         `yaml:"current-context"`
}

type NamedCluster struct {
	Name    string
	Cluster Cluster
}

type Cluster struct {
	Server                   string
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

type NamedUser struct {
	Name string
	User User
}

type User struct {
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
}

type NamedContext struct {
	Name    string
	Context Context
}

type Context struct {
	Cluster string
	User    string
}

// WriteKubeconfigFile creates a self-contained kubeconfig file with the given key pair data.
func WriteKubeconfigFile(fileSystem afero.Fs, path, apiEndpoint, caData, certData, keyData string) error {
	caDataBase64 := base64.StdEncoding.EncodeToString([]byte(caData))
	certDataBase64 := base64.StdEncoding.EncodeToString([]byte(certData))
	keyDataBase64 := base64.StdEncoding.EncodeToString([]byte(keyData))

	kc := Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []NamedCluster{
			NamedCluster{
				Name: "this-cluster",
				Cluster: Cluster{
					Server:                   apiEndpoint,
					CertificateAuthorityData: caDataBase64,
				},
			},
		},
		Users: []NamedUser{
			NamedUser{
				Name: "this-user",
				User: User{
					ClientCertificateData: certDataBase64,
					ClientKeyData:         keyDataBase64,
				},
			},
		},
		Contexts: []NamedContext{
			NamedContext{
				Name: "this-context",
				Context: Context{
					Cluster: "this-cluster",
					User:    "this-user",
				},
			},
		},
		CurrentContext: "this-context",
	}

	yamlBytes, err := yaml.Marshal(kc)
	if err != nil {
		return microerror.Mask(err)
	}

	err = afero.WriteFile(fileSystem, path, yamlBytes, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
