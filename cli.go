package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

const (
	bits               = 4096
	privateKeyFileName = "minecraft_rsa"
	publicKeyFileName  = "minecraft_rsa.pub"
	digOceanKeyName    = "minecraft"
	dropletName        = "minecraft"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println(helpText)
		return
	}
	command := os.Args[1]
	if command == "help" {
		fmt.Println(helpText)
		return
	}

	token := &TokenSource{AccessToken: readToken()}
	oauthClient := oauth2.NewClient(context.Background(), token)
	client := godo.NewClient(oauthClient)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fmt.Println(client.UserAgent)

	usr, err := user.Current()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sshDir := filepath.Join(usr.HomeDir, ".ssh")
	files, err := ioutil.ReadDir(sshDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(sshDir, 0700)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !contains(files, privateKeyFileName) && !contains(files, publicKeyFileName) {
		privKey, pubKey, err := rsaKeyPair(bits)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		privKeyPath := filepath.Join(sshDir, privateKeyFileName)
		pubKeyPath := filepath.Join(sshDir, publicKeyFileName)
		if err := writeKeyToFile(privKey, privKeyPath); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := writeKeyToFile(pubKey, pubKeyPath); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		_, _, err = client.Keys.Create(ctx, &godo.KeyCreateRequest{
			Name:      digOceanKeyName,
			PublicKey: string(pubKey),
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	Flag := flag.NewFlagSet("", flag.ExitOnError)
	switch command {
	case "start":
		droplets, _, err := client.Droplets.ListByTag(ctx, dropletName, &godo.ListOptions{})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, droplet := range droplets {
			if droplet.Name == dropletName {
				var ip string
				if ip, err = droplet.PublicIPv4(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				fmt.Printf("Instance already exist: %s\n", ip)
				os.Exit(0)
			}
		}
		world := Flag.String("world", "", "name of world folder to upload")
		serverProperties := Flag.String("server-properties", "", "comma separated key=value pairs of server properties")
		admins := Flag.String("admins", "", "comma separated list of admin usernames")
		whitelisted := Flag.String("whitelisted", "", "comma separated list of whitelisted usernames")
		Flag.Parse(os.Args[2:])
		fmt.Println("world has value: ", *world)
		fmt.Println("server properties: ", *serverProperties)
		fmt.Println(*admins)
		fmt.Println(*whitelisted)
		fmt.Println(os.Args)
		publicKeyPath := filepath.Join(sshDir, publicKeyFileName)
		publicKey, err := readPublicKey(publicKeyPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pubKeyFingerprint := ssh.FingerprintLegacyMD5(publicKey)
		ip, err := createDroplet(ctx, client, pubKeyFingerprint, dropletName, dropletName)
		fmt.Printf("Instance ip: %s", ip)
		return

	case "stop":
		//
	case "download":
		//
	default:
		fmt.Printf("'%s' is not a command, type 'mc help' for commands\n", command)
	}
}

func readPublicKey(filepath string) (ssh.PublicKey, error) {
	key, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	k, _, _, _, err := ssh.ParseAuthorizedKey(key)
	return k, err
}

func createDroplet(ctx context.Context, client *godo.Client, pubKeyFingerprint, name, tag string) (ip string, err error) {
	droplet, createResp, err := client.Droplets.Create(ctx, &godo.DropletCreateRequest{
		Name:   dropletName,
		Region: "ams3",
		Size:   "s-1vcpu-1gb",
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{
				Fingerprint: pubKeyFingerprint,
			},
		},
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-18-04-x64",
		},
		Tags: []string{dropletName},
		//UserData: startupscript
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// capture action and wait for 100% before returning ip
	defer createResp.Body.Close()
	//createResp.Body.Read()

	//ip, err := droplet.PublicIPv4()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(createResp)
}

func contains(files []os.FileInfo, filename string) bool {
	for _, file := range files {
		if file.Name() == filename {
			return true
		}
	}
	return false
}

func rsaKeyPair(bits int) (privKeyPEM, pubKey []byte, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	pubKey = ssh.MarshalAuthorizedKey(publicKey)
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privKey),
	}
	privKeyPEM = pem.EncodeToMemory(&privBlock)
	return privKeyPEM, pubKey, nil
}

func writeKeyToFile(keyBytes []byte, filepath string) error {
	err := ioutil.WriteFile(filepath, keyBytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

// readToken returns api token or terminates program
func readToken() (token string) {
	file, err := os.Open("token.txt")
	if os.IsNotExist(err) {
		fmt.Println("error: token.txt does not exist in current directory")
		os.Exit(1)
	} else if err != nil {
		fmt.Println(err)
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}
	return strings.TrimSpace(string(b))
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}
