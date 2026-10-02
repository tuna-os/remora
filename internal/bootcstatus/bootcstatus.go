// Package bootcstatus queries bootc status --json to extract image references and digests.
package bootcstatus

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// BootedImage returns the image ref the system is currently booted from,
// via bootc status --json.
func BootedImage() (string, error) {
	out, err := exec.Command("bootc", "status", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("bootc status (is this a bootc system?): %w", err)
	}
	return parseBootedImage(out)
}

func parseBootedImage(statusJSON []byte) (string, error) {
	var status struct {
		Status struct {
			Booted struct {
				Image struct {
					Image struct {
						Image string `json:"image"`
					} `json:"image"`
				} `json:"image"`
			} `json:"booted"`
		} `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return "", fmt.Errorf("parsing bootc status: %w", err)
	}
	ref := status.Status.Booted.Image.Image.Image
	if ref == "" {
		return "", fmt.Errorf("bootc status reports no booted image")
	}
	return ref, nil
}

// BootedImageDigest returns the image ref and digest the system is booted
// from. The digest is what makes a base pin reproducible: a tag moves, a
// digest does not.
func BootedImageDigest() (ref, digest string, err error) {
	out, err := exec.Command("bootc", "status", "--json").Output()
	if err != nil {
		return "", "", fmt.Errorf("bootc status (is this a bootc system?): %w", err)
	}
	return parseBootedImageDigest(out)
}

func parseBootedImageDigest(statusJSON []byte) (string, string, error) {
	var status struct {
		Status struct {
			Booted struct {
				Image struct {
					Image struct {
						Image     string `json:"image"`
						Transport string `json:"transport"`
					} `json:"image"`
					ImageDigest string `json:"imageDigest"`
				} `json:"image"`
			} `json:"booted"`
		} `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return "", "", fmt.Errorf("parsing bootc status: %w", err)
	}
	img := status.Status.Booted.Image
	if img.Image.Image == "" {
		return "", "", fmt.Errorf("bootc status reports no booted image")
	}
	return img.Image.Image, img.ImageDigest, nil
}

// StagedOrBootedDigest returns the digest of the image the system will next
// boot: the staged deployment if there is one, otherwise the booted one.
// remora compares a freshly built image against this to decide whether a
// switch would change anything.
func StagedOrBootedDigest() (string, error) {
	out, err := exec.Command("bootc", "status", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("bootc status: %w", err)
	}
	return parseStagedOrBootedDigest(out)
}

func parseStagedOrBootedDigest(statusJSON []byte) (string, error) {
	var status struct {
		Status struct {
			Staged *struct {
				Image struct {
					ImageDigest string `json:"imageDigest"`
				} `json:"image"`
			} `json:"staged"`
			Booted *struct {
				Image struct {
					ImageDigest string `json:"imageDigest"`
				} `json:"image"`
			} `json:"booted"`
		} `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return "", fmt.Errorf("parsing bootc status: %w", err)
	}
	if s := status.Status.Staged; s != nil && s.Image.ImageDigest != "" {
		return s.Image.ImageDigest, nil
	}
	if b := status.Status.Booted; b != nil {
		return b.Image.ImageDigest, nil
	}
	return "", nil
}
