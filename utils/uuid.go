package utils

import (
	"github.com/google/uuid"
	"strings"
)

const (
	ProUuidJoinChar = "-"
)

func GetProUuid(prefix ...string) (kisId string) {
	idStr := strings.Replace(uuid.New().String(), ProUuidJoinChar, "", -1)
	kisId = formatProID(idStr, prefix...)

	return
}

func formatProID(idStr string, prefix ...string) string {
	var kisId string

	for _, fix := range prefix {
		kisId += fix
		kisId += ProUuidJoinChar
	}

	kisId += idStr

	return kisId
}
