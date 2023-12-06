// This script applies three Linux specific fixes for Resonite:
//
//  1. FreeImage is named incorrectly, requiring a symlink
//  2. Brotli implementation is broken OOTB, causing sync issues and requiring a library rebuild + runtime files
//  3. On NVidia specifically,
//     "sRGB textures do not load due to an invalid format argument in a call to glTexSubImage2D()",
//     needing the infamous superpiss preload from iamgreaser, bless your patient soul
//
// The script has an implicit dependency on "dotnet"
// for building the Brotli fix and takes the absolute path to Resonite_Data as input

package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"github.com/adrg/xdg"
	"github.com/go-git/go-git/v5"
	cp "github.com/otiai10/copy"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	pathToResonite := os.Args[1]

	fmt.Println("Creating symbolic link for libFreeImage")
	err := symlinkLibFreeImage(pathToResonite)
	if err != nil {
		panic(err)
	}

	fmt.Println("Building and replacing Brotli dlls")
	FixBrotli(pathToResonite)

	fmt.Println("Provisioning Superpiss workaround")
	applySuperpiss()
}

func symlinkLibFreeImage(pathToResonite string) error {
	return os.Symlink(
		pathToResonite+"/Plugins/libFreeImage.so",
		pathToResonite+"/Plugins/libFreeimage.so",
	)
}

func FixBrotli(pathToResonite string) {
	const gitBuildPath = "/tmp/resonite-brotli-dotnet-shocktail39"
	err := os.RemoveAll(gitBuildPath)
	if err != nil {
		panic(err)
	}

	_, err = git.PlainClone(
		gitBuildPath, false, &git.CloneOptions{
			URL:      "https://github.com/shocktail39/resonite-brotli.net.git",
			Progress: os.Stdout,
		},
	)
	if err != nil {
		panic(err)
	}

	build := exec.Command(
		"dotnet", "publish", "-f", "net462", gitBuildPath+"/Brotli.NET/Brotli.Core/",
	)
	err = build.Run()
	if err != nil {
		panic(err)
	}

	// "Go is my favorite language because I love creating utilities from scratch that other languages offers OOTB."
	// - Abhijit Sarkar, https://stackoverflow.com/questions/10485743/contains-method-for-a-slice#comment125661061_70802740
	const publishResultFolder = gitBuildPath + "/Brotli.NET/Brotli.Core/bin/Debug/net462/publish/"

	r, err := os.Open(publishResultFolder + "Brotli.Core.dll")
	if err != nil {
		panic(err)
	}
	defer r.Close()
	brotlicoreBytes, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(pathToResonite+"/Managed/Brotli.Core.dll", brotlicoreBytes, 0750)
	if err != nil {
		panic(err)
	}

	err = cp.Copy(publishResultFolder+"runtimes", pathToResonite+"/Managed/runtimes")
	if err != nil {
		panic(err)
	}
}
func applySuperpiss() {
	const superpissDownload = "https://github.com/Yellow-Dog-Man/Resonite-Issues/files/12900246/bug-69-workaround.zip"

	resp, err := http.Get(superpissDownload)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		log.Fatal(err)
	}

	file, err := zipReader.File[1].Open()
	if err != nil {
		panic(err)
	}

	fileContents, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	err = os.Mkdir(xdg.DataHome+"/resonitefixes", 0750)
	if !errors.Is(err, fs.ErrExist) {
		panic(err)
	}

	err = os.WriteFile(xdg.DataHome+"/resonitefixes/superpiss.so", fileContents, 0750)
	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"\nIf you use Nvidia and you have issues launching, set this as your launch option: LD_PRELOAD=%s %%command%%",
		xdg.DataHome+"/resonitefixes/superpiss.so",
	)
}
