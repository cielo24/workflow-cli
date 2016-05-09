package cmd

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"github.com/deis/workflow-cli/controller/api"
	"github.com/deis/workflow-cli/controller/client"
	"github.com/deis/workflow-cli/controller/models/keys"
	"github.com/deis/workflow-cli/pkg/ssh"
)

// KeysList lists a user's keys.
func KeysList(results int) error {
	c, err := client.New()

	if err != nil {
		return err
	}

	if results == defaultLimit {
		results = c.ResponseLimit
	}

	keys, count, err := keys.List(c, results)

	if err != nil {
		return err
	}

	fmt.Printf("=== %s Keys%s", c.Username, limitCount(len(keys), count))

	for _, key := range keys {
		fmt.Printf("%s %s...%s\n", key.ID, key.Public[:16], key.Public[len(key.Public)-10:])
	}
	return nil
}

// KeyRemove removes keys.
func KeyRemove(keyID string) error {
	c, err := client.New()

	if err != nil {
		return err
	}

	fmt.Printf("Removing %s SSH Key...", keyID)

	if err = keys.Delete(c, keyID); err != nil {
		fmt.Println()
		return err
	}

	fmt.Println(" done")
	return nil
}

// KeyAdd adds keys.
func KeyAdd(keyLocation string) error {
	c, err := client.New()

	if err != nil {
		return err
	}

	var key api.KeyCreateRequest

	if keyLocation == "" {
		key, err = chooseKey()
	} else {
		key, err = getKey(keyLocation)
	}

	if err != nil {
		return err
	}

	fmt.Printf("Uploading %s to deis...", path.Base(key.Name))

	if _, err = keys.New(c, key.ID, key.Public); err != nil {
		fmt.Println()
		return err
	}

	fmt.Println(" done")
	return nil
}

func chooseKey() (api.KeyCreateRequest, error) {
	keys, err := listKeys()

	if err != nil {
		return api.KeyCreateRequest{}, err
	}

	fmt.Println("Found the following SSH public keys:")

	for i, key := range keys {
		fmt.Printf("%d) %s %s\n", i+1, path.Base(key.Name), key.ID)
	}

	fmt.Println("0) Enter path to pubfile (or use keys:add <key_path>)")

	var selected string

	fmt.Print("Which would you like to use with Deis? ")
	fmt.Scanln(&selected)

	numSelected, err := strconv.Atoi(selected)

	if err != nil {
		return api.KeyCreateRequest{}, err
	}

	if numSelected > len(keys)+1 {
		return api.KeyCreateRequest{}, fmt.Errorf("%d is not a valid option", numSelected)
	}

	if numSelected == 0 {
		var filename string

		fmt.Print("Enter the path to the pubkey file: ")
		fmt.Scanln(&filename)

		return getKey(filename)
	}

	return keys[numSelected-1], nil
}

func listKeys() ([]api.KeyCreateRequest, error) {
	folder := path.Join(client.FindHome(), ".ssh")
	files, err := ioutil.ReadDir(folder)

	if err != nil {
		return nil, err
	}

	var keys []api.KeyCreateRequest

	for _, file := range files {
		if path.Ext(file.Name()) == ".pub" {
			key, err := getKey(path.Join(folder, file.Name()))

			if err == nil {
				keys = append(keys, key)
			} else {
				fmt.Println(err)
			}
		}
	}

	return keys, nil
}

func getKey(filename string) (api.KeyCreateRequest, error) {
	keyContents, err := ioutil.ReadFile(filename)

	if err != nil {
		return api.KeyCreateRequest{}, err
	}

	backupID := strings.Split(path.Base(filename), ".")[0]
	keyInfo, err := ssh.ParsePubKey(backupID, keyContents)
	if err != nil {
		return api.KeyCreateRequest{}, fmt.Errorf("%s is not a valid ssh key", filename)
	}
	return api.KeyCreateRequest{ID: keyInfo.ID, Public: keyInfo.Public, Name: filename}, nil
}
