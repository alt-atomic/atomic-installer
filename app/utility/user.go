package utility

import (
	"installer/lib"
	"os/user"
)

// Константа максимальной длины имени пользователя.
const maxNameLen = 32

func isUsernameUsed(username string) bool {
	_, err := user.Lookup(username)
	return err == nil
}

func IsValidUsername(username string, parentalControlsEnabled bool) (bool, string) {
	var empty, inUse, tooLong, valid, parentalControlsConflict bool
	var tip string

	if username == "" {
		empty = true
		inUse = false
		tooLong = false
	} else {
		empty = false
		inUse = isUsernameUsed(username)
		tooLong = len(username) > maxNameLen
	}

	valid = true

	// Если имя не занято, не пустое и не слишком длинное, проверяем символы
	if !inUse && !empty && !tooLong {
		for i, r := range username {
			if i == 0 {
				if r < 'a' || r > 'z' {
					valid = false
				}
			} else {
				if !((r >= 'a' && r <= 'z') ||
					(r >= '0' && r <= '9') ||
					r == '_' || r == '-') {
					valid = false
				}
			}
		}
	}

	parentalControlsConflict = parentalControlsEnabled && username == "administrator"
	valid = !empty && !inUse && !tooLong && !parentalControlsConflict && valid
	if !empty && (inUse || tooLong || parentalControlsConflict || !valid) {
		if inUse {
			tip = lib.T_("Sorry, this username is already in use. Please try another one.")
		} else if tooLong {
			tip = lib.T_("The username is too long.")
		} else if len(username) > 0 && (username[0] < 'a' || username[0] > 'z') {
			tip = lib.T_("The username must start with a lowercase letter from a to z.")
		} else if parentalControlsConflict {
			tip = lib.T_("Sorry, this username is not available. Please try another one.")
		} else {
			tip = lib.T_("The username may only consist of lowercase letters (a-z), digits, and the characters '-' or '_'.")
		}
	} else {
		tip = lib.T_("This username will be used for creating your home directory and cannot be changed.")
	}

	return valid, tip
}
