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

package install

import (
	"fmt"
	"installer/lib"
	"sync"
)

// InstallerStatus — тип статуса установки.
type InstallerStatus int

const (
	StatusNotStarted = iota
	StatusCheckingEnvironment
	StatusDownloadImage
	StatusRemountingTmp
	StatusPreparingDisk
	StatusInstallingSystem
	StatusCreatedCommit
	StatusConfiguringSystem
	StatusFinalizingInstallation
	StatusCompleted
	StatusError
)

// SafeStatus — хранилище статуса с каналом уведомлений.
type SafeStatus struct {
	mu       sync.Mutex
	status   InstallerStatus
	progress string
	notify   chan struct{}
}

// NewSafeStatus создаёт новый SafeStatus с начальными настройками.
func NewSafeStatus() *SafeStatus {
	return &SafeStatus{
		status: StatusNotStarted,
		notify: make(chan struct{}, 1),
	}
}

// SetStatus безопасно устанавливает новый статус и уведомляет подписчиков.
func (s *SafeStatus) SetStatus(newStatus InstallerStatus) {
	s.mu.Lock()
	s.status = newStatus
	s.mu.Unlock()
	s.notifyChange()
}

// SetProgress обновляет строку прогресса.
func (s *SafeStatus) SetProgress(progress string) {
	s.mu.Lock()
	s.progress = progress
	s.mu.Unlock()
	s.notifyChange()
}

// GetStatusText возвращает текстовое описание текущего статуса установки с прогрессом.
func (s *SafeStatus) GetStatusText() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.status {
	case StatusCreatedCommit:
		return lib.T("Creating ostree repository")
	case StatusDownloadImage:
		return lib.T(fmt.Sprintf("Downloading image: %s", s.progress))
	case StatusNotStarted:
		return lib.T("Starting installation")
	case StatusCheckingEnvironment:
		return lib.T("Checking required commands and environment")
	case StatusRemountingTmp:
		return lib.T("Checking temporary directory")
	case StatusPreparingDisk:
		return lib.T("Preparing disk: cleaning, partitioning")
	case StatusInstallingSystem:
		return lib.T("Installing system")
	case StatusConfiguringSystem:
		return lib.T("Configuring system")
	case StatusFinalizingInstallation:
		return lib.T("Finalizing installation, verification and cleanup")
	case StatusCompleted:
		return lib.T("Installation completed successfully")
	case StatusError:
		return lib.T("Installation error")
	default:
		return lib.T("Unknown status")
	}
}

// GetStatus возвращает текущий статус установки.
func (s *SafeStatus) GetStatus() InstallerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// NotifyChan возвращает канал уведомлений об изменении статуса.
func (s *SafeStatus) NotifyChan() <-chan struct{} {
	return s.notify
}

func (s *SafeStatus) notifyChange() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}
