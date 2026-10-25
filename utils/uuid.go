package utils

import (
	"github.com/google/uuid"
	"strings"
)

const (
	ProUUIDJoinChar = "-"
)

func GetProUUID(prefix ...string) (kisID string) {
	idStr := strings.Replace(uuid.New().String(), ProUUIDJoinChar, "", -1)
	kisID = formatProID(idStr, prefix...)

	return
}

func formatProID(idStr string, prefix ...string) string {
	var kisID string

	for _, fix := range prefix {
		kisID += fix
		kisID += ProUUIDJoinChar
	}

	kisID += idStr

	return kisID
}
