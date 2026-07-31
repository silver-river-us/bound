package githubactivity

import "time"

type Source interface {
	Organizations() ([]Organization, error)
	Activities(organization string, since, until time.Time) ([]Activity, []string)
}
