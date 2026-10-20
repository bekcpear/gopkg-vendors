package tlsflag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradfitz/monogok/internal/config"
)

type ErrNotYetCreated struct {
	HostConfigPath string
	CertPath       string
	KeyPath        string
}

func (e *ErrNotYetCreated) Error() string {
	return "self-signed certificate not yet created"
}

func CertificatePathsFor(useTLS, hostname string) (certPath string, keyPath string, _ error) {
	hostConfigPath := config.HostnameSpecific(hostname)
	certPath = filepath.Join(string(hostConfigPath), "cert.pem")
	keyPath = filepath.Join(string(hostConfigPath), "key.pem")
	exist := true
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		exist = false
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		exist = false
	}

	switch useTLS {
	case "self-signed":
		if !exist {
			return "", "", &ErrNotYetCreated{
				HostConfigPath: string(hostConfigPath),
				CertPath:       certPath,
				KeyPath:        keyPath,
			}
		}

	case "off":
		return "", "", nil

	case "":
		if !exist {
			return "", "", nil
		}

	default:
		parts := strings.Split(useTLS, ",")
		certPath = parts[0]
		if len(parts) > 1 {
			keyPath = parts[1]
		} else {
			return "", "", fmt.Errorf("no private key supplied")
		}
	}
	return certPath, keyPath, nil
}
