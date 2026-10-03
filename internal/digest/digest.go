// Package digest handles image digest resolution and manipulation.
package digest

import (
	"fmt"
	"os/exec"
	"strings"
)

// PinBase returns ref pinned to a digest. A ref that already carries a
// digest is returned unchanged; otherwise the digest is resolved from the
// registry with skopeo.
func PinBase(ref string) (string, error) {
	if _, _, ok := SplitDigest(ref); ok {
		return ref, nil
	}
	digest, err := LatestDigest(ref)
	if err != nil {
		return "", err
	}
	return ref + "@" + digest, nil
}

// SplitDigest splits "image@sha256:..." into its name and digest. ok is
// false when ref carries no digest.
func SplitDigest(ref string) (name, digest string, ok bool) {
	i := strings.LastIndex(ref, "@")
	if i < 0 {
		return ref, "", false
	}
	return ref[:i], ref[i+1:], true
}

// LatestDigest asks the registry for the current digest behind ref.
func LatestDigest(ref string) (string, error) {
	name, _, _ := SplitDigest(ref)
	out, err := exec.Command("skopeo", "inspect", "--format", "{{.Digest}}", "docker://"+name).Output()
	if err != nil {
		return "", fmt.Errorf("skopeo inspect %s: %w", name, err)
	}
	digest := strings.TrimSpace(string(out))
	if digest == "" {
		return "", fmt.Errorf("skopeo returned an empty digest for %s", name)
	}
	return digest, nil
}

// LocalDigest returns the digest podman records for a local image ref, or
// "" if the image is not present locally.
func LocalDigest(ref string) (string, error) {
	out, err := exec.Command("podman", "inspect", "--format", "{{.Digest}}", ref).Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}
