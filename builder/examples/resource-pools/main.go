package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gen0cide/laforge/builder/vsphere"
)

func main() {
	httpClient := http.Client{}

	baseUrl, urlExists := os.LookupEnv("VSPHERE_URL")
	username, usernameExists := os.LookupEnv("VSPHERE_USERNAME")
	password, passwordExists := os.LookupEnv("VSPHERE_PASSWORD")
	if !urlExists || !usernameExists || !passwordExists {
		log.Fatalf("please set VSPHERE_URL (exists? %t), VSPHERE_USERNAME (exists? %t), and VSPHERE_PASSWORD (exists? %t)", urlExists, usernameExists, passwordExists)
	}
	vshpere := vsphere.VSphere{
		Client: httpClient,
		BaseUrl: baseUrl,	
		Username: username,
		Password: password,
	}

	resourcePools, err := vshpere.ListResourcePools()
	if err != nil {
		log.Fatalf("error while getting resource pools: %v", err)
	}
	for _, resourcePool := range resourcePools {
		fmt.Printf("%s [%s]\n", resourcePool.Name, resourcePool.Identifier)
	}
}