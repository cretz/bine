package process

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cretz/bine/util"
)

func ControlPortFromFileContents(contents string) (int, error) {
	contents = strings.TrimSpace(contents)
	_, port, ok := util.PartitionString(contents, ':')
	if !ok || !strings.HasPrefix(contents, "PORT=") {
		return 0, fmt.Errorf("Invalid port format: %v", contents)
	}
	return strconv.Atoi(port)
}
