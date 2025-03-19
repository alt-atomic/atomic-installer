// Atomic Installer
// Copyright (C) 2025 Дмитрий Удалов dmitry@udalov.online
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package utility

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type IPInfoResponse struct {
	IP       string `json:"ip"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
}

func GetTimeZoneFromIP() (string, error) {
	url := fmt.Sprintf("https://ipinfo.io/json")

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("incorrect status: %d", resp.StatusCode)
	}

	var ipInfo IPInfoResponse
	if err = json.NewDecoder(resp.Body).Decode(&ipInfo); err != nil {
		return "", fmt.Errorf("incorrect answwer: %v", err)
	}

	return ipInfo.Timezone, nil
}
