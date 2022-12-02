package main

import (
	"os/user"
	"strconv"
)

// CurrentUser performs the horrendous extraction of userID and groupID from
// the multiplatform user.User to Linux-specific numbers. Will fail on
// platforms where Uid or Gid are not numbers. Will misbehave if Uid or Gid are
// not unsigned 32-bit integers.
func CurrentUser() (uid, gid uint32, err error) {
	userInfo, err := user.Current()
	if err != nil {
		return 0, 0, err
	}

	u, err := strconv.Atoi(userInfo.Uid)
	if err != nil {
		return 0, 0, err
	}

	g, err := strconv.Atoi(userInfo.Gid)
	if err != nil {
		return 0, 0, err
	}

	return uint32(u), uint32(g), nil
}
