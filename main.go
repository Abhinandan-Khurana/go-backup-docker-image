package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type Config struct {
	BackupDir    string
	MaxWorkers   int
	Verbose      bool
	CompressType string
}

// ImageInfo stores metadata about backed up images
type ImageInfo struct {
	ImageName    string    `json:"image_name"`
	ImageID      string    `json:"image_id"`
	Tags         []string  `json:"tags"`
	Size         int64     `json:"size"`
	BackupDate   time.Time `json:"backup_date"`
	CompressType string    `json:"compress_type"`
}

var config Config

var banner = `
               _             _                    _         _               _                     
  __ _ ___ ___| |__  __ _ __| |___  _ _ __ ___ __| |___  __| |_____ _ _ ___(_)_ __  __ _ __ _ ___ 
 / _' / _ \___| '_ \/ _' / _| / / || | '_ \___/ _' / _ \/ _| / / -_) '_|___| | '  \/ _' / _' / -_)
 \__, \___/   |_.__/\__,_\__|_\_\\_,_| .__/   \__,_\___/\__|_\_\___|_|     |_|_|_|_\__,_\__, \___|
 |___/                               |_|                                                |___/     

~ A Docker Image Backup Tool ~
Made with ❤️ by Abhinandan-Khurana
`

func main() {
	config = Config{
		BackupDir:    "docker-backups",
		MaxWorkers:   3,
		Verbose:      false,
		CompressType: "gzip",
	}

	rootCmd := &cobra.Command{
		Use:   "go-backup-docker-image",
		Short: "Docker Image Backup Tool",
		Long:  "A tool to backup Docker images as tarballs and restore them when needed",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Name() != "help" && cmd.Name() != "completion" {
				color.New(color.FgCyan, color.Bold).Println(banner)
			}
		},
	}

	backupCmd := &cobra.Command{
		Use:   "backup [IMAGE_NAME...]",
		Short: "Backup Docker images as tarballs",
		Run:   runBackup,
	}
	backupCmd.Flags().StringVarP(&config.BackupDir, "dir", "d", config.BackupDir, "Directory to store backups")
	backupCmd.Flags().IntVarP(&config.MaxWorkers, "workers", "w", config.MaxWorkers, "Maximum number of concurrent workers")
	backupCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", config.Verbose, "Enable verbose logging")
	backupCmd.Flags().StringVarP(&config.CompressType, "compress", "c", config.CompressType, "Compression type (gzip, none)")
	backupCmd.Flags().StringP("file", "f", "", "Read image names from file")
	backupCmd.Flags().BoolP("stdin", "s", false, "Read image names from stdin")

	restoreCmd := &cobra.Command{
		Use:   "restore [TARBALL_PATH...]",
		Short: "Restore Docker images from tarballs",
		Run:   runRestore,
	}
	restoreCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", config.Verbose, "Enable verbose logging")
	restoreCmd.Flags().StringP("file", "f", "", "Read tarball paths from file")
	restoreCmd.Flags().BoolP("stdin", "s", false, "Read tarball paths from stdin")
	restoreCmd.Flags().IntVarP(&config.MaxWorkers, "workers", "w", config.MaxWorkers, "Maximum number of concurrent workers")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available backup images",
		Run:   runList,
	}
	listCmd.Flags().StringVarP(&config.BackupDir, "dir", "d", config.BackupDir, "Backup directory to list")
	listCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", config.Verbose, "Show detailed information")

	rootCmd.AddCommand(backupCmd, restoreCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		color.New(color.FgRed, color.Bold).Println(err)
		os.Exit(1)
	}
}

func runBackup(cmd *cobra.Command, args []string) {
	var imageNames []string

	fileInput, _ := cmd.Flags().GetString("file")
	stdInput, _ := cmd.Flags().GetBool("stdin")

	// If stdin flag is used, read image names from stdin
	if stdInput {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			name := strings.TrimSpace(scanner.Text())
			if name != "" {
				imageNames = append(imageNames, name)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdin: %v", err)
		}
	} else if fileInput != "" {
		// If file flag is used, read image names from file
		file, err := os.Open(fileInput)
		if err != nil {
			log.Fatalf("Error opening file %s: %v", fileInput, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			name := strings.TrimSpace(scanner.Text())
			if name != "" {
				imageNames = append(imageNames, name)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
	} else {
		imageNames = args
	}

	if len(imageNames) == 0 {
		log.Fatal("No image names provided. Use command arguments, --file, or --stdin")
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	// Initialize Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.MaxWorkers)
	ctx := context.Background()

	for _, imageName := range imageNames {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(img string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			backupImage(cli, ctx, img)
		}(imageName)
	}

	wg.Wait()
	fmt.Println("All backup operations completed")
}

// backupImage creates a tarball backup of a single Docker image
func backupImage(cli *client.Client, ctx context.Context, imageName string) {
	if config.Verbose {
		fmt.Printf("Starting backup of image: %s\n", imageName)
	}

	img, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		log.Printf("Error inspecting image %s: %v", imageName, err)
		return
	}

	safeImageName := strings.ReplaceAll(imageName, "/", "_")
	safeImageName = strings.ReplaceAll(safeImageName, ":", "_")
	timestamp := time.Now().Format("20060102-150405")
	tarballName := filepath.Join(config.BackupDir, fmt.Sprintf("%s-%s.tar", safeImageName, timestamp))

	if config.CompressType == "gzip" {
		tarballName += ".gz"
	}

	var cmd *exec.Cmd

	if config.CompressType == "gzip" {
		fmt.Printf("Saving image %s to %s (gzip compressed)...\n", imageName, tarballName)
		cmd = exec.Command("sh", "-c", fmt.Sprintf("docker save %s | gzip > %s", imageName, tarballName))
	} else {
		fmt.Printf("Saving image %s to %s...\n", imageName, tarballName)
		cmd = exec.Command("docker", "save", "-o", tarballName, imageName)
	}

	if config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to save image %s: %v", imageName, err)
		return
	}

	imageInfo := ImageInfo{
		ImageName:    imageName,
		ImageID:      img.ID,
		Tags:         img.RepoTags,
		Size:         img.Size,
		BackupDate:   time.Now(),
		CompressType: config.CompressType,
	}

	metadataPath := tarballName + ".json"
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		log.Printf("Failed to create metadata file for %s: %v", imageName, err)
		return
	}
	defer metadataFile.Close()

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(imageInfo); err != nil {
		log.Printf("Failed to write metadata for %s: %v", imageName, err)
		return
	}

	color.New(color.FgGreen, color.Bold).Printf("Successfully backed up image %s to %s\n", imageName, tarballName)
}

func runRestore(cmd *cobra.Command, args []string) {
	var tarballPaths []string

	fileInput, _ := cmd.Flags().GetString("file")
	stdInput, _ := cmd.Flags().GetBool("stdin")

	if stdInput {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			path := strings.TrimSpace(scanner.Text())
			if path != "" {
				tarballPaths = append(tarballPaths, path)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdin: %v", err)
		}
	} else if fileInput != "" {
		file, err := os.Open(fileInput)
		if err != nil {
			log.Fatalf("Error opening file %s: %v", fileInput, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			path := strings.TrimSpace(scanner.Text())
			if path != "" {
				tarballPaths = append(tarballPaths, path)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
	} else {
		tarballPaths = args
	}

	if len(tarballPaths) == 0 {
		log.Fatal("No tarball paths provided. Use command arguments, --file, or --stdin")
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.MaxWorkers)

	for _, tarballPath := range tarballPaths {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(path string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			restoreImage(path)
		}(tarballPath)
	}

	wg.Wait()
	color.New(color.FgGreen, color.Bold).Println("All restore operations completed")
}

func restoreImage(tarballPath string) {
	if config.Verbose {
		color.New(color.FgBlue, color.Bold).Printf("Starting restore of image from: %s\n", tarballPath)
	}

	// Check for metadata file to determine compression type
	metadataPath := tarballPath + ".json"
	var compressed bool

	// First check the extension
	if strings.HasSuffix(tarballPath, ".tar.gz") || strings.HasSuffix(tarballPath, ".tgz") {
		compressed = true
	} else {
		// Then check metadata if available
		if _, err := os.Stat(metadataPath); err == nil {
			metadataFile, err := os.Open(metadataPath)
			if err == nil {
				defer metadataFile.Close()

				var imageInfo ImageInfo
				decoder := json.NewDecoder(metadataFile)
				if err := decoder.Decode(&imageInfo); err == nil {
					compressed = imageInfo.CompressType == "gzip"
				}
			}
		}
	}

	var cmd *exec.Cmd

	if compressed {
		color.New(color.FgYellow, color.Bold).Printf("Loading compressed image from %s...\n", tarballPath)
		cmd = exec.Command("sh", "-c", fmt.Sprintf("gunzip -c %s | docker load", tarballPath))
	} else {
		fmt.Printf("Loading image from %s...\n", tarballPath)
		cmd = exec.Command("docker", "load", "-i", tarballPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to load image from %s: %v\n%s", tarballPath, err, output)
		return
	}

	fmt.Printf("Successfully restored image from %s\n", tarballPath)
	fmt.Printf("Docker output: %s\n", output)
}

func runList(cmd *cobra.Command, args []string) {
	if _, err := os.Stat(config.BackupDir); os.IsNotExist(err) {
		color.New(color.FgRed, color.Bold).Printf("Backup directory %s does not exist\n", config.BackupDir)
		return
	}

	files, err := os.ReadDir(config.BackupDir)
	if err != nil {
		log.Fatalf("Failed to read backup directory: %v", err)
	}

	tarFiles := make(map[string]os.FileInfo)
	metaFiles := make(map[string]ImageInfo)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		info, err := file.Info()
		if err != nil {
			continue
		}

		if strings.HasSuffix(name, ".tar") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") {
			tarFiles[name] = info
		} else if strings.HasSuffix(name, ".json") {
			// Try to parse metadata
			metaPath := filepath.Join(config.BackupDir, name)
			metaFile, err := os.Open(metaPath)
			if err == nil {
				defer metaFile.Close()

				var imageInfo ImageInfo
				decoder := json.NewDecoder(metaFile)
				if err := decoder.Decode(&imageInfo); err == nil {
					baseName := strings.TrimSuffix(name, ".json")
					metaFiles[baseName] = imageInfo
				}
			}
		}
	}

	if len(tarFiles) == 0 {
		color.New(color.FgHiRed, color.Bold).Println("No backups found")
		return
	}

	color.New(color.FgHiBlue, color.Bold).Println("Available Docker image backups:")
	fmt.Println("---------------------------------")

	for name, info := range tarFiles {
		fmt.Printf("Backup: %s\n", name)
		fmt.Printf("  Size: %.2f MB\n", float64(info.Size())/(1024*1024))
		fmt.Printf("  Date: %s\n", info.ModTime().Format(time.RFC3339))

		// Display metadata if available
		if meta, exists := metaFiles[name]; exists {
			fmt.Printf("  Image: %s\n", meta.ImageName)
			fmt.Printf("  Tags: %s\n", strings.Join(meta.Tags, ", "))
			if config.Verbose {
				fmt.Printf("  ID: %s\n", meta.ImageID)
				fmt.Printf("  Compression: %s\n", meta.CompressType)
			}
		}
		fmt.Println()
	}
}
