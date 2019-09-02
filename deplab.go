package deplab

import (
	"bytes"
	"fmt"
	"github.com/pivotal/deplab/docker"
	"github.com/pivotal/deplab/metadata"
	"github.com/pivotal/deplab/outputs"
	"github.com/pivotal/deplab/providers"
	"log"
	"os/exec"
	"strings"
)

func Run(inputImageTar, inputImage, gitPath, tag, outputImageTar, metadataFilePath, dpkgFilePath string) {
	if inputImageTar != "" {
		stdout, stderr, err := runCommand("docker", "load", "-i", inputImageTar)
		if err != nil {
			log.Fatalf("could not load docker image from tar: %s", stderr)
		}

		imageTag := ""
		if strings.Contains(stdout.String(), "image ID") {
			imageTag = strings.Trim(stdout.String(), "Loaded image ID: \n")
		} else {
			imageTag = strings.Trim(stdout.String(), "Loaded image: \n")
		}
		inputImage = imageTag
	}
	dependencies, err := generateDependencies(inputImage, gitPath)
	if err != nil {
		log.Fatalf("error generating dependencies: %s", err)
	}
	md := metadata.Metadata{Dependencies: dependencies}

	md.Base = providers.BuildOSMetadata(inputImage)

	resp, err := docker.CreateNewImage(inputImage, md, tag)
	if err != nil {
		log.Fatalf("could not create new image: %s\n", err)
	}

	newID, err := docker.GetIDOfNewImage(resp)
	if err != nil {
		log.Fatalf("could not get ID of the new image: %s\n", err)
	}

	fmt.Println(newID)

	writeOutputs(md, metadataFilePath, dpkgFilePath)

	if outputImageTar != "" {
		id := newID

		if tag != "" {
			id = tag
		}

		_, stderr, err := runCommand("docker", "save", id, "-o", outputImageTar)
		if err != nil {
			log.Fatalf("could not save docker image to tar: %s", stderr)
		}
	}

}

func generateDependencies(imageName, pathToGit string) ([]metadata.Dependency, error) {
	var dependencies []metadata.Dependency

	dpkgList, err := providers.BuildDebianDependencyMetadata(imageName)
	if err != nil {
		log.Fatalf("debian package metadata: %s", err)
	}
	if dpkgList.Type != "" {
		dependencies = append(dependencies, dpkgList)
	}

	gitMetadata, err := providers.BuildGitDependencyMetadata(pathToGit)
	if err != nil {
		log.Fatalf("git metadata: %s", err)
	}
	dependencies = append(dependencies, gitMetadata)

	return dependencies, nil
}

func writeOutputs(md metadata.Metadata, metadataFilePath, dpkgFilePath string) {
	if metadataFilePath != "" {
		outputs.WriteMetadataFile(md, metadataFilePath)
	}

	if dpkgFilePath != "" {
		outputs.WriteDpkgFile(md, dpkgFilePath)
	}
}

func runCommand(cmd string, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	dockerLoad := exec.Command(cmd, args...)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	dockerLoad.Stdout = stdout
	dockerLoad.Stderr = stderr

	err := dockerLoad.Run()
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}
