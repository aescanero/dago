package usecase

import (
	"fmt"
	"regexp"

	"github.com/aescanero/dago/libs/domain"
)

var semverRE = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func validateSemver(version string) error {
	if !semverRE.MatchString(version) {
		return fmt.Errorf("%w: version must match \\d+\\.\\d+\\.\\d+", domain.ErrValidation)
	}
	return nil
}
