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
	"bufio"
	"context"
	"fmt"
	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
	"installer/app/utility"
	"installer/lib"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// InstallerService — сервис
type InstallerService struct {
	data   InstallerData
	Status *SafeStatus
}

// NewInstallerService — конструктор сервиса
func NewInstallerService(installerData InstallerData) *InstallerService {
	return &InstallerService{
		data:   installerData,
		Status: NewSafeStatus(),
	}
}

type User struct {
	Login    string
	Password string
}

type InstallerData struct {
	Image              string
	Disk               string
	TypeFilesystem     string
	TypeBoot           string
	IsCryptoFilesystem bool
	LuksPassword       string
	User               User
}

const containerDir = "/var/lib/containers"

var timezone = "Europe/Moscow"

func (i *InstallerService) RunInstall() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	i.Status.SetStatus(StatusCheckingEnvironment)
	go i.checkTimeZone()

	i.Status.SetStatus(StatusRemountingTmp)
	i.checkAndRemountTmp()

	if err := i.prepareDisk(ctx); err != nil {
		i.Status.SetStatus(StatusError)
		lib.Log.Errorf("Disk preparation error: %v", err)
		return
	}

	if err := i.installToFilesystem(ctx); err != nil {
		i.Status.SetStatus(StatusError)
		lib.Log.Errorf("Installation error: %v", err)
		return
	}

	partitions, err := i.getNamedPartitionsWithCrypto()
	if err != nil {
		i.Status.SetStatus(StatusError)
		lib.Log.Errorf("Error obtaining named partitions: %v", err)
		return
	}

	if err = i.cleanupTemporaryPartition(ctx, partitions); err != nil {
		i.Status.SetStatus(StatusError)
		lib.Log.Errorf("Temporary partition cleanup error: %v", err)
		return
	}

	i.Status.SetStatus(StatusCompleted)
	lib.Log.Info("Installation completed successfully!")
}

func (i *InstallerService) checkAndRemountTmp() {
	var stat syscall.Statfs_t

	if err := syscall.Statfs("/tmp", &stat); err != nil {
		lib.Log.Errorf("Error reading /tmp statistics: %v", err)
		return
	}

	// Calculate partition size in gigabytes
	total := float64(stat.Blocks*uint64(stat.Bsize)) / (1 << 30)
	lib.Log.Infof("Current /tmp size: %.2f GB", total)

	// If less than 5 GB, attempt to remount /tmp
	if total < 5.0 {
		cmd := exec.Command("mount", "-o", "remount,size=5G", "/tmp")
		output, err := cmd.CombinedOutput()
		if err != nil {
			lib.Log.Errorf("Error remounting /tmp: %v (output: %s)", err, string(output))
			return
		}
		lib.Log.Info("Successfully remounted /tmp, command output:", string(output))
	} else {
		lib.Log.Info("The /tmp size is sufficient, remounting is not required.")
	}
}

func (i *InstallerService) checkTimeZone() {
	ipTimeZone, err := utility.GetTimeZoneFromIP()
	if err != nil {
		lib.Log.Error(err.Error())
		return
	}

	timezone = ipTimeZone
}

func (i *InstallerService) cleanupTemporaryPartition(ctx context.Context, partitions map[string]PartitionInfo) error {
	i.Status.SetStatus(StatusFinalizingInstallation)
	lib.Log.Info("Удаление временного раздела и расширение root-раздела...")

	// Размонтируем временный раздел
	lib.Log.Infof("Размонтирование временного раздела %s...", partitions["temp"].Path)
	if err := i.unmount(ctx, containerDir); err != nil {
		return fmt.Errorf("ошибка размонтирования временного раздела: %v", err)
	}

	// Размонтируем временный раздел
	lib.Log.Infof("Размонтирование временного раздела %s...", "/var/tmp")
	if err := i.unmount(ctx, "/var/tmp"); err != nil {
		logrus.Errorf("ошибка размонтирования временного раздела: %v", err)
	}

	// Удаляем временный раздел
	lib.Log.Infof("Удаление временного раздела %s...", partitions["temp"].Path)
	cmd := exec.Command("parted", "-s", i.data.Disk, "rm", partitions["temp"].Number)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка удаления временного раздела: %v", err)
	}

	// Определяем физический раздел для resize (исходный или зашифрованный)
	var resizeTarget string
	if i.data.IsCryptoFilesystem && partitions["root"].OriginalPath != "" {
		resizeTarget = partitions["root"].OriginalPath
		lib.Log.Infof("Расширение LUKS раздела %s до 100%%...", resizeTarget)
	} else {
		resizeTarget = partitions["root"].Path
		lib.Log.Infof("Расширение root-раздела %s до 100%%...", resizeTarget)
	}

	cmd = exec.Command("parted", "-s", i.data.Disk, "resizepart", partitions["root"].Number, "100%")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка изменения размера раздела: %v", err)
	}

	// Для LUKS разделов нужно расширить и сам зашифрованный том
	if i.data.IsCryptoFilesystem && partitions["root"].OriginalPath != "" {
		lib.Log.Infof("Расширение LUKS тома...")
		resizeCmd := exec.Command("cryptsetup", "resize", "cryptroot")
		resizeCmd.Stdout = os.Stdout
		resizeCmd.Stderr = os.Stderr
		resizeCmd.Stdin = strings.NewReader(i.data.LuksPassword)
		if err := resizeCmd.Run(); err != nil {
			return fmt.Errorf("ошибка расширения LUKS тома: %v", err)
		}

		// Задержка и принудительное обновление размера устройства
		time.Sleep(3 * time.Second)
		exec.Command("udevadm", "settle").Run()
		exec.Command("partprobe").Run()
	}

	// Проверяем тип файловой системы root-раздела
	lib.Log.Infof("Проверка типа файловой системы раздела %s...", partitions["root"].Path)
	cmd = exec.Command("blkid", "-o", "value", "-s", "TYPE", partitions["root"].Path)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ошибка проверки типа файловой системы: %v", err)
	}

	fsType := strings.TrimSpace(string(output))
	lib.Log.Infof("Тип файловой системы: %s", fsType)

	if fsType == "btrfs" {
		// Для btrfs используем btrfs filesystem resize
		mountPoint := "/mnt/btrfs-root"
		lib.Log.Infof("Изменение размера файловой системы btrfs на разделе %s...", partitions["root"].Path)

		// Монтируем раздел
		if err = i.mountDisk(partitions["root"].Path, mountPoint, ""); err != nil {
			return fmt.Errorf("ошибка монтирования btrfs-раздела: %v", err)
		}
		defer i.unmountDisk(mountPoint) // Размонтируем после завершения

		// Выполняем resize на точке монтирования
		cmd = exec.Command("btrfs", "filesystem", "resize", "max", mountPoint)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка изменения размера файловой системы btrfs: %v", err)
		}
	} else if fsType == "ext4" {
		// Для ext4 используем resize2fs
		cmd = exec.Command("e2fsck", "-f", "-y", partitions["root"].Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		lib.Log.Infof("Проверка и исправление файловой системы ext4 на разделе %s...", partitions["root"].Path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка проверки файловой системы ext4: %v", err)
		}

		lib.Log.Infof("Изменение размера файловой системы ext4 на разделе %s...", partitions["root"].Path)
		cmd = exec.Command("resize2fs", partitions["root"].Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка изменения размера файловой системы ext4: %v", err)
		}
	} else {
		return fmt.Errorf("неподдерживаемая файловая система: %s", fsType)
	}

	lib.Log.Info("Временный раздел удалён, root-раздел расширен.")
	return nil
}

// isMounted проверяет, примонтирован ли путь
func (i *InstallerService) isMounted(path string) bool {
	cmd := exec.Command("mountpoint", "-q", path)
	err := cmd.Run()
	return err == nil
}

// unmount размонтирует путь, если он примонтирован
func (i *InstallerService) unmount(ctx context.Context, path string) error {
	if i.isMounted(path) {
		lib.Log.Infof("Размонтирование %s...", path)
		fmt.Printf("Размонтирование %s...", path)
		cmd := exec.CommandContext(ctx, "umount", "-l", path)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка размонтирования %s: %v", path, err)
		}
		lib.Log.Infof("%s успешно размонтирован.", path)
	}
	return nil
}

func (i *InstallerService) freeDisk() {
	lib.Log.Infof("Освобождение диска %s от процессов", i.data.Disk)
	killCmd := exec.Command("fuser", "-km", i.data.Disk)
	if _, err := killCmd.CombinedOutput(); err != nil {
		lib.Log.Warningf("Не найдены активные процессы для %s: %v", i.data.Disk, err)
	}

	exec.Command("sync").Run()
}

// prepareDisk выполняет подготовку диска
func (i *InstallerService) prepareDisk(ctx context.Context) error {
	i.Status.SetStatus(StatusPreparingDisk)
	paths := []string{"/mnt/target/boot/efi", "/mnt/target/boot", containerDir, "/mnt/target", "/var/tmp"}

	for _, path := range paths {
		_ = i.unmount(ctx, path)
	}
	i.freeDisk()

	lib.Log.Infof("Подготовка диска %s с файловой системой %s в режиме %s", i.data.Disk, i.data.TypeFilesystem, i.data.TypeBoot)

	// Команды для разметки
	var commands [][]string

	if i.data.TypeBoot == "LEGACY" {
		commands = [][]string{
			{"wipefs", "--all", i.data.Disk},
			{"parted", "-s", i.data.Disk, "mklabel", "gpt"},
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "1MiB", "3MiB"},                               // BIOS Boot Partition (2 МиБ)
			{"parted", "-s", i.data.Disk, "set", "1", "bios_grub", "on"},                                     // BIOS Boot Partition
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "fat32", "3MiB", "1003MiB"},                   // EFI раздел (1 ГБ)
			{"parted", "-s", i.data.Disk, "set", "2", "boot", "on"},                                          // EFI раздел
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "ext4", "1003MiB", "3003MiB"},                 // Boot раздел (2 ГБ)
			{"parted", "-s", i.data.Disk, "mkpart", "primary", i.data.TypeFilesystem, "3003MiB", "25000MiB"}, // Root раздел
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "ext4", "25000MiB", "60000MiB"},               // Временный раздел
		}
	} else if i.data.TypeBoot == "UEFI" {
		commands = [][]string{
			{"wipefs", "--all", i.data.Disk},
			{"parted", "-s", i.data.Disk, "mklabel", "gpt"},
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "fat32", "1MiB", "601MiB"},                    // EFI раздел (600 МБ)
			{"parted", "-s", i.data.Disk, "set", "1", "boot", "on"},                                          // EFI раздел
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "ext4", "601MiB", "2601MiB"},                  // Boot раздел (2 ГБ)
			{"parted", "-s", i.data.Disk, "mkpart", "primary", i.data.TypeFilesystem, "2601MiB", "25000MiB"}, // Root раздел
			{"parted", "-s", i.data.Disk, "mkpart", "primary", "ext4", "25000MiB", "60000MiB"},               // Временный раздел
		}
	} else {
		return fmt.Errorf("неизвестный тип загрузки: %s", i.data.TypeBoot)
	}

	for _, args := range commands {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ошибка выполнения команды %s: %v", args[0], err)
		}
	}

	partitions, err := i.getNamedPartitions()
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	if len(partitions) < 3 {
		return fmt.Errorf("недостаточно разделов на диске")
	}

	var partitionList []string
	for key, value := range partitions {
		partitionList = append(partitionList, fmt.Sprintf("%s: %s", key, value))
	}
	lib.Log.Infof("Partitions: %s", strings.Join(partitionList, ", "))

	// LUKS шифрование root раздела
	if i.data.IsCryptoFilesystem {
		lib.Log.Infof("Настройка LUKS шифрования для раздела %s...", partitions["root"].Path)

		originalRootPath := partitions["root"].Path

		// Форматируем с LUKS2, передаем пароль через stdin
		cryptsetupCmd := exec.CommandContext(ctx, "cryptsetup", "luksFormat", "--type", "luks2", "--batch-mode", "--force-password", originalRootPath)
		cryptsetupCmd.Stdout = os.Stdout
		cryptsetupCmd.Stderr = os.Stderr
		cryptsetupCmd.Stdin = strings.NewReader(i.data.LuksPassword)
		if err := cryptsetupCmd.Run(); err != nil {
			return fmt.Errorf("ошибка создания LUKS раздела: %v", err)
		}

		// Открываем зашифрованный раздел
		openCmd := exec.CommandContext(ctx, "cryptsetup", "luksOpen", originalRootPath, "cryptroot")
		openCmd.Stdout = os.Stdout
		openCmd.Stderr = os.Stderr
		openCmd.Stdin = strings.NewReader(i.data.LuksPassword)
		if err := openCmd.Run(); err != nil {
			return fmt.Errorf("ошибка открытия LUKS раздела: %v", err)
		}

		// Создаем новую map с обновленными путями
		rootInfo := partitions["root"]
		rootInfo.OriginalPath = originalRootPath
		rootInfo.Path = "/dev/mapper/cryptroot"
		partitions["root"] = rootInfo
	}

	formats := []struct {
		cmd  string
		args []string
	}{
		{"mkfs.vfat", []string{"-F32", partitions["efi"].Path}}, // Форматирование EFI раздела
		{"mkfs.ext4", []string{partitions["boot"].Path}},        // Форматирование boot раздела
	}

	if i.data.TypeFilesystem == "ext4" {
		formats = append(formats, struct {
			cmd  string
			args []string
		}{"mkfs.ext4", []string{partitions["root"].Path}})

	} else if i.data.TypeFilesystem == "btrfs" {
		formats = append(formats, struct {
			cmd  string
			args []string
		}{"mkfs.btrfs", []string{"-f", partitions["root"].Path}})
	} else {
		return fmt.Errorf("неизвестная файловая система: %s", i.data.TypeFilesystem)
	}

	formats = append(formats, struct {
		cmd  string
		args []string
	}{"mkfs.ext4", []string{partitions["temp"].Path}})

	for _, format := range formats {
		cmd := exec.CommandContext(ctx, format.cmd, format.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("ошибка форматирования %s: %v", format.args[0], err)
		}
	}

	if i.data.TypeFilesystem == "btrfs" {
		if err = i.createBtrfsSubVolumes(partitions["root"].Path); err != nil {
			return fmt.Errorf("ошибка создания подтомов Btrfs: %v", err)
		}
	}

	// Создание временного раздела
	tempCommands := [][]string{
		{"mkdir", "-p", containerDir},
		{"mount", partitions["temp"].Path, containerDir},
	}

	for _, args := range tempCommands {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("ошибка выполнения команды %s: %v", args[0], err)
		}
	}
	lib.Log.Infof("Диск %s успешно подготовлен.", i.data.Disk)

	return nil
}

func (i *InstallerService) createBtrfsSubVolumes(rootPartition string) error {
	mountPoint := "/mnt/btrfs-setup"
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}
	defer os.RemoveAll(mountPoint)

	if err := i.mountDisk(rootPartition, mountPoint, "rw,subvol=/"); err != nil {
		return fmt.Errorf("ошибка монтирования Btrfs раздела: %v", err)
	}
	defer i.unmountDisk(mountPoint)

	subVolumes := []string{"@", "@home", "@var"}
	for _, subVol := range subVolumes {
		subVolPath := fmt.Sprintf("%s/%s", mountPoint, subVol)
		if _, err := os.Stat(subVolPath); os.IsNotExist(err) {
			cmd := exec.Command("btrfs", "subvolume", "create", subVolPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ошибка создания подтома %s: %v", subVol, err)
			}
		} else {
			lib.Log.Warning("Подтом %s уже существует, пропуск.", subVol)
		}
	}

	return nil
}

// installToFilesystem выполняет установку с использованием bootc
func (i *InstallerService) installToFilesystem(ctx context.Context) error {
	i.Status.SetStatus(StatusInstallingSystem)

	mountPoint := "/mnt/target"
	mountBtrfsVar := "/mnt/btrfs/var"
	mountBtrfsHome := "/mnt/btrfs/home"
	mountPointBoot := "/mnt/target/boot"
	efiMountPoint := "/mnt/target/boot/efi"

	// Получаем именованные разделы (с учетом LUKS если активен)
	partitions, err := i.getNamedPartitionsWithCrypto()
	if err != nil {
		return fmt.Errorf("ошибка получения разделов: %v", err)
	}

	// Монтируем разделы
	if i.data.TypeFilesystem == "btrfs" {
		if err = i.mountDisk(partitions["root"].Path, mountPoint, "subvol=@"); err != nil {
			return fmt.Errorf("ошибка монтирования корневого подтома: %v", err)
		}
	} else {
		if err = i.mountDisk(partitions["root"].Path, mountPoint, ""); err != nil {
			return fmt.Errorf("ошибка монтирования root раздела: %v", err)
		}
	}

	if err = i.mountDisk(partitions["boot"].Path, mountPointBoot, ""); err != nil {
		return fmt.Errorf("ошибка монтирования boot раздела: %v", err)
	}

	if err = i.mountDisk(partitions["efi"].Path, efiMountPoint, ""); err != nil {
		return fmt.Errorf("ошибка монтирования EFI раздела: %v", err)
	}

	// Выполняем установку с использованием bootc
	installCmd := i.buildBootcCommand(partitions)

	cmd := exec.CommandContext(ctx, "podman", "run", "--rm", "--privileged", "--pid=host",
		"--security-opt", "label=type:unconfined_t",
		"-v", containerDir+":/var/lib/containers",
		"-v", "/dev:/dev",
		"-v", "/mnt/target:/mnt/target",
		"--security-opt", "label=disable",
		i.data.Image,
		"sh", "-c", installCmd,
	)

	tmpDir := containerDir + "/tmp"

	if err = os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// /var/tmp → /var/lib/containers/tmp
	cmdMount := exec.Command("mount", "--bind", tmpDir, "/var/tmp")
	if output, err := cmdMount.CombinedOutput(); err != nil {
		return fmt.Errorf("bind mount failed: %v\n%s", err, string(output))
	}

	// Устанавливаем переменную окружения для поддержки TTY.
	env := os.Environ()
	env = append(env, "TERM=xterm-256color")
	cmd.Env = env

	lib.Log.Infof("Запущен процесс загрузки и установки образа")
	i.Status.SetStatus(StatusDownloadImage)
	// Запускаем команду через pty
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to run command from pty: %v", err)
	}
	defer func() { _ = ptmx.Close() }()

	// Устанавливаем размер терминала (опционально)
	if err = pty.Setsize(ptmx, &pty.Winsize{
		Rows: 40,
		Cols: 120,
	}); err != nil {
		return err
	}

	progressRegex := regexp.MustCompile(`Copying blob\s+\S+\s+\[.*?\]\s+([\d\.]+[A-Za-z]+)\s*/\s*([\d\.]+[A-Za-z]+)\s*\|\s*([\d\.]+\s*[A-Za-z/]+)`)

	scanner := bufio.NewScanner(ptmx)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 100*1024*1024)

	// Хотим хранить только последние 5 строк для ошибки
	const maxLines = 5
	linesBuffer := make([]string, 0, maxLines)

	var lastUpdate time.Time
	updateInterval := 500 * time.Millisecond
	for scanner.Scan() {
		line := scanner.Text()
		matches := progressRegex.FindStringSubmatch(line)
		linesBuffer = append(linesBuffer, line)
		if len(linesBuffer) > maxLines {
			linesBuffer = linesBuffer[len(linesBuffer)-maxLines:]
		}

		fmt.Println(line)
		if len(matches) == 4 {
			downloaded := matches[1]
			total := matches[2]
			speed := matches[3]
			progressMsg := fmt.Sprintf("%s из %s, %s", downloaded, total, speed)
			now := time.Now()
			if now.Sub(lastUpdate) >= updateInterval {
				i.Status.SetProgress(progressMsg)
				lastUpdate = now
			}
		} else if strings.Contains(line, "/sysroot/ostree/repo") {
			i.Status.SetStatus(StatusCreatedCommit)
		} else if strings.Contains(line, "Initializing ostree layout") {
			i.Status.SetStatus(StatusInstallingSystem)
		}
	}

	if err = scanner.Err(); err != nil {
		lib.Log.Errorf("error reading output: %v", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf(
			"error install: %s",
			strings.Join(linesBuffer, "\n"),
		)
	}

	i.unmountDisk(efiMountPoint)
	i.unmountDisk(mountPointBoot)
	i.unmountDisk(mountPoint)

	i.Status.SetStatus(StatusConfiguringSystem)
	var ostreeDeployPath string
	if i.data.TypeFilesystem == "btrfs" {
		if err = i.mountDisk(partitions["root"].Path, mountPoint, "rw,subvol=@"); err != nil {
			return fmt.Errorf("ошибка повторного монтирования корневого подтома: %v", err)
		}

		if err = i.mountDisk(partitions["root"].Path, mountBtrfsVar, "subvol=@var"); err != nil {
			return fmt.Errorf("ошибка монтирования подтома @var: %v", err)
		}

		if err = i.mountDisk(partitions["root"].Path, mountBtrfsHome, "subvol=@home"); err != nil {
			return fmt.Errorf("ошибка монтирования подтома @home: %v", err)
		}

		ostreeDeployPath, err = i.findOstreeDeployPath(mountPoint)
		if err != nil {
			return fmt.Errorf("ошибка поиска ostree deploy пути: %v", err)
		}

		if err = i.configureUserAndRoot(ostreeDeployPath, i.data.User.Login, i.data.User.Password); err != nil {
			return fmt.Errorf("ошибка настройки пользователя и root: %v", err)
		}

		if err = i.configureTimezone(ostreeDeployPath, timezone); err != nil {
			return fmt.Errorf("ошибка установки timezone: %v", err)
		}

		if err = i.configureHostname(ostreeDeployPath, "atomic"); err != nil {
			return fmt.Errorf("ошибка установки timezone: %v", err)
		}

		// Копируем содержимое /var в подтом @var
		if err = i.copyWithRsync(fmt.Sprintf("%s/var/", ostreeDeployPath), mountBtrfsVar); err != nil {
			return fmt.Errorf("ошибка копирования /var в @var: %v", err)
		}

		// Копируем содержимое /home в подтом @home
		if err = i.copyWithRsync(fmt.Sprintf("%s/home/", ostreeDeployPath), mountBtrfsHome); err != nil {
			return fmt.Errorf("ошибка копирования /home в @home: %v", err)
		}

		//Очищаем содержимое /var внутри ostree
		if err = i.clearDirectory(fmt.Sprintf("%s/var", ostreeDeployPath)); err != nil {
			return fmt.Errorf("ошибка очистки содержимого /var: %v", err)
		}

		//путь к папке var
		varDeployPath := fmt.Sprintf("%s/var", filepath.Join(ostreeDeployPath, "../../"))

		//Очищаем содержимое ostree/deploy/default/var
		if err = i.clearDirectory(varDeployPath); err != nil {
			return fmt.Errorf("ошибка очистки содержимого /ostree/deploy/default/var: %v", err)
		}

		selabeledFilePath := fmt.Sprintf("%s/.ostree-selabeled", varDeployPath)
		lib.Log.Infof("Создание файла %s", selabeledFilePath)

		file, err := os.Create(selabeledFilePath)
		if err != nil {
			return fmt.Errorf("ошибка создания файла .ostree-selabeled: %v", err)
		}

		errFile := file.Close()
		if errFile != nil {
			return fmt.Errorf("ошибка очистки содержимого /ostree/deploy/default/var: %v", errFile)
		}
	} else {
		if err = i.mountDisk(partitions["root"].Path, mountPoint, "rw"); err != nil {
			return fmt.Errorf("ошибка повторного монтирования root раздела: %v", err)
		}

		ostreeDeployPath, err = i.findOstreeDeployPath(mountPoint)
		if err != nil {
			return fmt.Errorf("ошибка поиска ostree deploy пути: %v", err)
		}

		if err = i.configureUserAndRoot(ostreeDeployPath, i.data.User.Login, i.data.User.Password); err != nil {
			return fmt.Errorf("ошибка настройки пользователя и root: %v", err)
		}

		varDeployPath := filepath.Join(ostreeDeployPath, "../../var/home")

		// Копируем содержимое /home из коммита внутрь varDeployPath
		if err = i.copyWithRsync(fmt.Sprintf("%s/home/", ostreeDeployPath), varDeployPath); err != nil {
			return fmt.Errorf("ошибка копирования /home в @home: %v", err)
		}

		// Очищаем содержимое /var внутри ostree
		if err = i.clearDirectory(fmt.Sprintf("%s/var", ostreeDeployPath)); err != nil {
			return fmt.Errorf("ошибка очистки содержимого /var: %v", err)
		}

		if err = i.configureTimezone(ostreeDeployPath, timezone); err != nil {
			return fmt.Errorf("ошибка установки timezone: %v", err)
		}

		if err = i.configureHostname(ostreeDeployPath, "atomic"); err != nil {
			return fmt.Errorf("ошибка установки timezone: %v", err)
		}
	}

	if err = i.mountDisk(partitions["boot"].Path, mountPointBoot, "rw"); err != nil {
		return fmt.Errorf("ошибка повторного монтирования boot раздела: %v", err)
	}

	if err = i.mountDisk(partitions["efi"].Path, efiMountPoint, "rw"); err != nil {
		return fmt.Errorf("ошибка повторного монтирования EFI раздела: %v", err)
	}

	// Генерация fstab
	lib.Log.Infof("Генерация fstab...")
	if err = i.generateFstab(mountPoint, partitions, i.data.TypeFilesystem); err != nil {
		return fmt.Errorf("ошибка генерации fstab: %v", err)
	}

	i.unmountDisk(efiMountPoint)
	i.unmountDisk(mountPointBoot)
	if i.data.TypeFilesystem == "btrfs" {
		i.unmountDisk(mountBtrfsHome)
		i.unmountDisk(mountBtrfsVar)
	}
	time.Sleep(5 * time.Second)
	i.unmountDisk(mountPoint)
	return nil
}

// buildBootcCommand создает команду bootc с флагами для LUKS
func (i *InstallerService) buildBootcCommand(partitions map[string]PartitionInfo) string {
	baseCmd := []string{"[ -f /usr/libexec/init-ostree.sh ] && /usr/libexec/init-ostree.sh; bootc install to-filesystem --skip-fetch-check --disable-selinux"}

	if i.data.TypeBoot != "UEFI" {
		baseCmd = append(baseCmd, "--generic-image")
	}

	if i.data.IsCryptoFilesystem {
		// UUID boot раздела
		bootUUID := i.getUUID(partitions["boot"].Path)
		baseCmd = append(baseCmd, fmt.Sprintf("--boot-mount-spec=UUID=%s", bootUUID))

		// Путь к зашифрованному разделу
		baseCmd = append(baseCmd, "--root-mount-spec=/dev/mapper/cryptroot")

		// UUID исходного раздела для LUKS
		originalPath := partitions["root"].OriginalPath
		if originalPath == "" {
			originalPath = partitions["root"].Path
		}
		rootUUID := i.getUUID(originalPath)
		baseCmd = append(baseCmd, fmt.Sprintf("--karg=rd.luks.name=%s=cryptroot", rootUUID))

		// Дополнительные флаги для btrfs
		if i.data.TypeFilesystem == "btrfs" {
			baseCmd = append(baseCmd, "--karg=rootflags=subvol=@")
		}
	}

	// Добавляем параметры для красивого экрана загрузки и Plymouth
	baseCmd = append(baseCmd, "--karg=rhgb")
	baseCmd = append(baseCmd, "--karg=quiet")
	baseCmd = append(baseCmd, "--karg=splash")
	baseCmd = append(baseCmd, "--karg=plymouth.enable=1")
	baseCmd = append(baseCmd, "--karg=rd.plymouth=1")

	baseCmd = append(baseCmd, "/mnt/target")
	return strings.Join(baseCmd, " ")
}

// configureHostname задаёт имя хоста через chroot
func (i *InstallerService) configureHostname(rootPath, hostname string) error {
	hostnameFile := filepath.Join(rootPath, "etc", "hostname")
	if err := os.WriteFile(hostnameFile, []byte(hostname+"\n"), 0644); err != nil {
		return fmt.Errorf("ошибка создания /etc/hostname: %v", err)
	}

	// Обновляем /etc/hosts
	hostsFile := filepath.Join(rootPath, "etc", "hosts")
	hostsContent := fmt.Sprintf("127.0.0.1 localhost %s\n::1 localhost %s\n", hostname, hostname)

	// Читаем существующий hosts если есть
	if existingHosts, err := os.ReadFile(hostsFile); err == nil {
		lines := strings.Split(string(existingHosts), "\n")
		var filteredLines []string
		for _, line := range lines {
			if !strings.Contains(line, "127.0.0.1") && !strings.Contains(line, "::1") && strings.TrimSpace(line) != "" {
				filteredLines = append(filteredLines, line)
			}
		}
		hostsContent = hostsContent + strings.Join(filteredLines, "\n") + "\n"
	}

	if err := os.WriteFile(hostsFile, []byte(hostsContent), 0644); err != nil {
		return fmt.Errorf("ошибка обновления /etc/hosts: %v", err)
	}

	lib.Log.Infof("Hostname %s успешно настроен в /etc/hostname и /etc/hosts", hostname)
	return nil
}

// configureTimezone устанавливает тайм-зону в указанном chroot окружении
func (i *InstallerService) configureTimezone(rootPath string, timezone string) error {
	lib.Log.Infof("Настройка таймзоны: %s", timezone)
	localtimePath := fmt.Sprintf("%s/etc/localtime", rootPath)

	// Удаляем существующий символический линк или файл
	if _, err := os.Lstat(localtimePath); err == nil {
		if err := os.Remove(localtimePath); err != nil {
			return fmt.Errorf("ошибка удаления старого localtime: %v", err)
		}
	}

	tzLink := fmt.Sprintf("/usr/share/zoneinfo/%s", timezone)
	cmd := exec.Command("ln", "-sf", tzLink, localtimePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка создания ссылки на таймзону: %v", err)
	}

	lib.Log.Infof("Таймзона %s успешно настроена.", timezone)
	return nil
}

func (i *InstallerService) configureUserAndRoot(rootPath string, userName string, password string) error {
	chrootCmd := func(args ...string) *exec.Cmd {
		cmd := exec.Command("chroot", append([]string{rootPath}, args...)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd
	}

	varHomePath := fmt.Sprintf("%s/var/home", rootPath)
	homeDir := fmt.Sprintf("/var/home/%s", userName)

	lib.Log.Infof("Проверка существования каталога /var/home...")
	if _, err := os.Stat(varHomePath); os.IsNotExist(err) {
		lib.Log.Warningf("Каталог %s не существует. Создаём...", varHomePath)
		if err = os.MkdirAll(varHomePath, 0755); err != nil {
			return fmt.Errorf("ошибка создания каталога %s: %v", varHomePath, err)
		}
	}

	lib.Log.Infof("Добавление пользователя...")
	cmd := chrootCmd("adduser", "-m", "-d", fmt.Sprintf("/var/home/%s", userName), "-G", "wheel", userName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка добавления пользователя %s: %v", userName, err)
	}

	lib.Log.Infof("Установка пароля пользователя...")
	cmd = chrootCmd("sh", "-c", fmt.Sprintf("echo '%s:%s' | chpasswd", userName, password))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка установки пароля для пользователя %s: %v", userName, err)
	}

	lib.Log.Infof("Установка пароля root...")
	cmd = chrootCmd("sh", "-c", fmt.Sprintf("echo 'root:%s' | chpasswd", password))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка установки пароля для root: %v", err)
	}

	lib.Log.Infof("Копирование файлов skel...")
	cmd = chrootCmd(
		"sh", "-c",
		fmt.Sprintf("[ -d /etc/skel ] && cp -r /etc/skel/. %s/", homeDir),
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка копирования skel: %v", err)
	}

	cmd = chrootCmd("chown", "-R", fmt.Sprintf("%s:%s", userName, userName), homeDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка изменения владельца: %v", err)
	}

	lib.Log.Infof("Пользователь и root настроены успешно.")
	return nil
}

func (i *InstallerService) clearDirectory(path string) error {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("ошибка чтения содержимого директории %s: %v", path, err)
	}

	for _, entry := range dirEntries {
		entryPath := fmt.Sprintf("%s/%s", path, entry.Name())

		if err = os.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("ошибка удаления %s: %v", entryPath, err)
		}
	}

	return nil
}

// copyWithRsync копирование с использованием команды rsync
func (i *InstallerService) copyWithRsync(src string, dst string) error {
	cmd := exec.Command("rsync", "-aHAX", src, dst)
	cmd.Stdout = nil
	cmd.Stderr = nil

	lib.Log.Infof("Копирование с использованием rsync: %s -> %s", src, dst)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка выполнения rsync: %v", err)
	}
	return nil
}

// findOstreeDeployPath находит путь к папке, заканчивающейся на .0
func (i *InstallerService) findOstreeDeployPath(mountPoint string) (string, error) {
	deployPath := fmt.Sprintf("%s/ostree/deploy/default/deploy", mountPoint)
	entries, err := os.ReadDir(deployPath)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения директории %s: %v", deployPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".0") {
			return fmt.Sprintf("%s/%s", deployPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("не найдена папка, в %s", deployPath)
}

func (i *InstallerService) generateFstab(mountPoint string, partitions map[string]PartitionInfo, rootFileSystem string) error {
	ostreeDeployPath, err := i.findOstreeDeployPath(mountPoint)
	if err != nil {
		return fmt.Errorf("ошибка поиска ostree deploy пути: %v", err)
	}
	fstabPath := fmt.Sprintf("%s/etc/fstab", ostreeDeployPath)

	lib.Log.Infof("Генерация %s...", fstabPath)

	fstabContent := "# Auto generate fstab from atomic-installer installer\n"

	if rootFileSystem == "btrfs" {
		fstabContent += fmt.Sprintf(
			"UUID=%s / btrfs subvol=@,compress=zstd:1,x-systemd.device-timeout=0 0 0\n",
			i.getUUID(partitions["root"].Path),
		)
		fstabContent += fmt.Sprintf(
			"UUID=%s /home btrfs subvol=@home,compress=zstd:1,x-systemd.device-timeout=0 0 0\n",
			i.getUUID(partitions["root"].Path),
		)
		fstabContent += fmt.Sprintf(
			"UUID=%s /var btrfs subvol=@var,compress=zstd:1,x-systemd.device-timeout=0 0 0\n",
			i.getUUID(partitions["root"].Path),
		)
	} else if rootFileSystem == "ext4" {
		fstabContent += fmt.Sprintf(
			"UUID=%s / ext4 defaults 1 1\n",
			i.getUUID(partitions["root"].Path),
		)
	} else {
		return fmt.Errorf("неизвестная файловая система: %s", rootFileSystem)
	}

	fstabContent += fmt.Sprintf(
		"UUID=%s /boot ext4 defaults 1 2\n",
		i.getUUID(partitions["boot"].Path),
	)
	fstabContent += fmt.Sprintf(
		"UUID=%s /boot/efi vfat umask=0077,shortname=winnt 0 2\n",
		i.getUUID(partitions["efi"].Path),
	)

	file, err := os.Create(fstabPath)
	if err != nil {
		return fmt.Errorf("ошибка создания %s: %v", fstabPath, err)
	}
	defer file.Close()

	if _, err = file.WriteString(fstabContent); err != nil {
		return fmt.Errorf("ошибка записи в %s: %v", fstabPath, err)
	}

	lib.Log.Infof("Файл %s успешно создан.", fstabPath)

	// Создание crypttab для LUKS разделов
	if i.data.IsCryptoFilesystem {
		if err := i.generateCrypttab(ostreeDeployPath, partitions); err != nil {
			return fmt.Errorf("ошибка создания crypttab: %v", err)
		}
	}

	return nil
}

func (i *InstallerService) generateCrypttab(ostreeDeployPath string, partitions map[string]PartitionInfo) error {
	crypttabPath := fmt.Sprintf("%s/etc/crypttab", ostreeDeployPath)
	lib.Log.Infof("Генерация %s...", crypttabPath)

	var originalPath string
	if partitions["root"].OriginalPath != "" {
		originalPath = partitions["root"].OriginalPath
	} else {
		originalPath = partitions["root"].Path
	}

	crypttabContent := fmt.Sprintf("cryptroot UUID=%s none luks\n", i.getUUID(originalPath))

	file, err := os.Create(crypttabPath)
	if err != nil {
		return fmt.Errorf("ошибка создания %s: %v", crypttabPath, err)
	}
	defer file.Close()

	if _, err = file.WriteString(crypttabContent); err != nil {
		return fmt.Errorf("ошибка записи в %s: %v", crypttabPath, err)
	}

	lib.Log.Infof("Файл %s успешно создан. Содержимое: %s", crypttabPath, crypttabContent)
	return nil
}

type PartitionInfo struct {
	Path         string
	Number       string
	OriginalPath string
}

func (i *InstallerService) getNamedPartitions() (map[string]PartitionInfo, error) {
	partitions, err := i.getPartitions(i.data.Disk)
	if err != nil {
		return nil, err
	}

	lib.Log.Infof("Список разделов:")
	for i, partition := range partitions {
		lib.Log.Infof("Раздел %d: %s", i+1, partition)
	}
	if i.data.TypeBoot == "LEGACY" && len(partitions) < 4 {
		return nil, fmt.Errorf("недостаточно разделов на диске для режима LEGACY")
	} else if i.data.TypeBoot == "UEFI" && len(partitions) < 3 {
		return nil, fmt.Errorf("недостаточно разделов на диске для режима UEFI")
	}

	// Карта с информацией о разделах
	namedPartitions := make(map[string]PartitionInfo)

	if i.data.TypeBoot == "LEGACY" {
		namedPartitions["bios"] = PartitionInfo{Path: partitions[0], Number: "1"} // BIOS Boot Partition
		namedPartitions["efi"] = PartitionInfo{Path: partitions[1], Number: "2"}  // EFI Partition
		namedPartitions["boot"] = PartitionInfo{Path: partitions[2], Number: "3"} // Boot Partition
		namedPartitions["root"] = PartitionInfo{Path: partitions[3], Number: "4"} // Root Partition
		namedPartitions["temp"] = PartitionInfo{Path: partitions[4], Number: "5"} // Temporary Partition
	} else if i.data.TypeBoot == "UEFI" {
		namedPartitions["efi"] = PartitionInfo{Path: partitions[0], Number: "1"}  // EFI Partition
		namedPartitions["boot"] = PartitionInfo{Path: partitions[1], Number: "2"} // Boot Partition
		namedPartitions["root"] = PartitionInfo{Path: partitions[2], Number: "3"} // Root Partition
		namedPartitions["temp"] = PartitionInfo{Path: partitions[3], Number: "4"} // Temporary Partition
	}

	return namedPartitions, nil
}

// getNamedPartitionsWithCrypto возвращает разделы с учетом LUKS шифрования
func (i *InstallerService) getNamedPartitionsWithCrypto() (map[string]PartitionInfo, error) {
	namedPartitions, err := i.getNamedPartitions()
	if err != nil {
		return nil, err
	}

	// Если используется LUKS, обновляем путь root раздела
	if i.data.IsCryptoFilesystem {
		rootInfo := namedPartitions["root"]
		rootInfo.OriginalPath = rootInfo.Path
		rootInfo.Path = "/dev/mapper/cryptroot"
		namedPartitions["root"] = rootInfo
	}

	return namedPartitions, nil
}

// getPartitionNames возвращает список всех разделов на указанном диске
func (i *InstallerService) getPartitions(disk string) ([]string, error) {
	cmd := exec.Command("lsblk", "-ln", "-o", "NAME,TYPE", disk)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения lsblk: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var partitions []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "part" { // Проверяем, что это раздел
			partitions = append(partitions, "/dev/"+fields[0])
		}
	}

	return partitions, nil
}

// mountDisk монтирует указанный раздел в точку монтирования
func (i *InstallerService) mountDisk(disk string, mountPoint string, options string) error {
	fmt.Printf("Монтирование диска %s в %s с опциями '%s'", disk, mountPoint, options)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("ошибка создания точки монтирования: %v", err)
	}
	var args []string
	if options != "" {
		args = append(args, "-o", options)
	}
	args = append(args, disk, mountPoint)
	cmd := exec.Command("mount", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка монтирования диска: %v", err)
	}
	return nil
}

// unmountDisk размонтирует указанную точку монтирования
func (i *InstallerService) unmountDisk(mountPoint string) {
	lib.Log.Infof("Размонтирование %s...", mountPoint)
	cmd := exec.Command("umount", mountPoint)
	if err := cmd.Run(); err != nil {
		lib.Log.Warningf("Ошибка размонтирования %s: %v", mountPoint, err.Error())
	}
}

// getUUID возвращает UUID указанного раздела
func (i *InstallerService) getUUID(disk string) string {
	cmd := exec.Command("blkid", "-s", "UUID", "-o", "value", disk)
	output, err := cmd.Output()
	if err != nil {
		lib.Log.Errorf("Ошибка получения UUID для %s: %v", disk, err)
		return ""
	}
	return strings.TrimSpace(string(output))
}
