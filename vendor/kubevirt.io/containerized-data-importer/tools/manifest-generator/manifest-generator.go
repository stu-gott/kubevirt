//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"flag"
	"os"
	"text/template"

	"github.com/golang/glog"

	cdicluster "kubevirt.io/containerized-data-importer/pkg/operator/resources/cluster"
	cdinamespaced "kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
)

type templateData struct {
	DockerRepo        string
	DockerTag         string
	ControllerImage   string
	ImporterImage     string
	ClonerImage       string
	APIServerImage    string
	UploadProxyImage  string
	UploadServerImage string
	OperatorImage     string
	Verbosity         string
	PullPolicy        string
	Namespace         string
}

var (
	dockerRepo        = flag.String("docker-repo", "", "")
	dockertag         = flag.String("docker-tag", "", "")
	controllerImage   = flag.String("controller-image", "", "")
	importerImage     = flag.String("importer-image", "", "")
	clonerImage       = flag.String("cloner-image", "", "")
	apiServerImage    = flag.String("apiserver-image", "", "")
	uploadProxyImage  = flag.String("uploadproxy-image", "", "")
	uploadServerImage = flag.String("uploadserver-image", "", "")
	operatorImage     = flag.String("operator-image", "", "")
	verbosity         = flag.String("verbosity", "1", "")
	pullPolicy        = flag.String("pull-policy", "", "")
	namespace         = flag.String("namespace", "", "")
)

func main() {
	templFile := flag.String("template", "", "")
	codeGroup := flag.String("code-group", "everything", "")
	flag.Parse()

	if *templFile != "" {
		generateFromFile(*templFile)
		return
	}

	generateFromCode(*codeGroup)
}

func generateFromFile(templFile string) {
	data := &templateData{
		Verbosity:         *verbosity,
		DockerRepo:        *dockerRepo,
		DockerTag:         *dockertag,
		ControllerImage:   *controllerImage,
		ImporterImage:     *importerImage,
		ClonerImage:       *clonerImage,
		APIServerImage:    *apiServerImage,
		UploadProxyImage:  *uploadProxyImage,
		UploadServerImage: *uploadServerImage,
		OperatorImage:     *operatorImage,
		PullPolicy:        *pullPolicy,
		Namespace:         *namespace,
	}

	file, err := os.OpenFile(templFile, os.O_RDONLY, 0)
	if err != nil {
		glog.Fatalf("Failed to open file %s: %v\n", templFile, err)
	}
	defer file.Close()

	tmpl := template.Must(template.ParseFiles(templFile))
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		glog.Fatalf("Error executing template: %v\n", err)
	}
}

func generateFromCode(codeGroup string) {
	var resources []interface{}

	crs, err := getClusterResources(codeGroup)
	if err != nil {
		glog.Fatalf("Error getting cluster resources: %v\n", err)
	}

	resources = append(resources, crs...)

	nsrs, err := getNamespacedResources(codeGroup)
	if err != nil {
		glog.Fatalf("Error getting namespaced resources: %v\n", err)
	}

	resources = append(resources, nsrs...)

	for _, resource := range resources {
		err = MarshallObject(resource, os.Stdout)
		if err != nil {
			glog.Fatalf("Error marshalling resource: %v\n", err)
		}
	}
}

// getClusterResources creates cluster-scoped resources for a specific group/component
// returning interface{} because caller is only interested in marshalling to json
// and is stuffing cluster-scomed and namespaced resources in the same slice
func getClusterResources(codeGroup string) ([]interface{}, error) {
	var result []interface{}
	var resources []cdicluster.Resource
	var err error

	args := &cdicluster.FactoryArgs{
		Namespace: *namespace,
	}

	if codeGroup == "everything" {
		resources, err = cdicluster.CreateAllResources(args)
	} else {
		resources, err = cdicluster.CreateResourceGroup(codeGroup, args)
	}

	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		result = append(result, resource.(interface{}))
	}

	return result, nil
}

// getNamespacedResources creates namespace-scoped resources for a specific group/component
// returning interface{} because caller is only interested in marshalling to json
// and is stuffing cluster-scomed and namespaced resources in the same slice
func getNamespacedResources(codeGroup string) ([]interface{}, error) {
	var result []interface{}
	var resources []cdinamespaced.Resource
	var err error

	args := &cdinamespaced.FactoryArgs{
		Verbosity:         *verbosity,
		DockerRepo:        *dockerRepo,
		DockerTag:         *dockertag,
		ControllerImage:   *controllerImage,
		ImporterImage:     *importerImage,
		ClonerImage:       *clonerImage,
		APIServerImage:    *apiServerImage,
		UploadProxyImage:  *uploadProxyImage,
		UploadServerImage: *uploadServerImage,
		PullPolicy:        *pullPolicy,
		Namespace:         *namespace,
	}

	if codeGroup == "everything" {
		resources, err = cdinamespaced.CreateAllResources(args)
	} else {
		resources, err = cdinamespaced.CreateResourceGroup(codeGroup, args)
	}

	if err != nil {
		return nil, err
	}

	for _, resource := range resources {
		result = append(result, resource.(interface{}))
	}

	return result, nil
}
