package main

import (
    "os"
    "log"
    "fmt"
    "sync"
    "bufio"
    "sort"
    "os/user"
    "os/exec"
    "path/filepath"
)

// babyface is a recon tool for subdomain and port scanning
// target hostnames.  It doesn't do anything special, it just
// wraps existing awesome tools.

// runs amass and subfinder together to find subdomains
// output saves to ~/.babyface/targets/<domain>/subdomains.txt
// then runs nmap across the subdomains to find ports and services
// output saves to ~/.babyface/targets/<domain>/nmap.txt

// utility functions

func expand(path string) (string, error) {
    if len(path) == 0 || path[0] != '~' {
        return path, nil
    }

    usr, err := user.Current()
    if err != nil {
        return "", err
    }
    return filepath.Join(usr.HomeDir, path[1:]), nil
}

func touch(path string) {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0666)
    }
}

func isCommandAvailable(name string) bool {
      cmd := exec.Command("command", "-v", name)
      if err := cmd.Run(); err != nil {
              return false
      }
      return true
}

func goInstall(repo string) {
    cmd := exec.Command("go", "get", repo)
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
    cmd = exec.Command("go", "install", "-i", repo)
    err = cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func readLines(filePath string) (lines []string, err error) {
    f, err := os.Open(filePath)
    if err != nil {
        return
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
          lines = append(lines, scanner.Text())
    }
    err = scanner.Err()
    return
}

func writeLines(path string, lines []string) error {
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()

    w := bufio.NewWriter(file)
    for _, line := range lines {
        fmt.Fprintln(w, line)
    }
    return w.Flush()
}

func removeDuplicatesUnordered(elements []string) []string {
    encountered := map[string]bool{}

    // Create a map of all unique elements.
    for v:= range elements {
        encountered[elements[v]] = true
    }

    // Place all keys from the map into a slice.
    result := []string{}
    for key, _ := range encountered {
        result = append(result, key)
    }
    return result
}


func uniqSort(path string) {
    // uniq and sort a given txt file
    lines, err := readLines(path)
    if err != nil {
        log.Fatal(err)
    }
    lines = removeDuplicatesUnordered(lines)
    sort.Strings(lines)
    err = writeLines(path, lines)
    if err != nil {
        log.Fatal(err)
    }
}

// main logic functions

func Setup(hostname string) {
    // setup ~/.babyface
    path, _ := expand("~/.babyface")
    targetsPath := filepath.Join(path, "targets")
    targetHostPath := filepath.Join(targetsPath, hostname)
    files := []string{path, targetsPath, targetHostPath}
    for _, fpath := range files {
        if _, err := os.Stat(fpath); os.IsNotExist(err) {
            _ = os.Mkdir(fpath, os.ModePerm)
        }
    }

    // todo: install subfinder, amass, and nmap if they aren't already
    if !isCommandAvailable("subfinder") {
        goInstall("github.com/subfinder/subfinder")
    }
    if !isCommandAvailable("amass") {
        goInstall("github.com/OWASP/Amass")
    }
}

func Subfinder(hostname, subdomainsPath string) {
    fmt.Println("subfinder -d", hostname, "-o", subdomainsPath)
    cmd := exec.Command("subfinder", "-d", hostname, "-o", subdomainsPath)
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func Amass(hostname, subdomainsPath string) {
    fmt.Println("amass -d", hostname, "-o", subdomainsPath, "-brute")
    cmd := exec.Command("amass", "-d", hostname, "-o", subdomainsPath, "-brute")
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func SubdomainScan(hostname, subdomainsPath string) {
    touch(subdomainsPath)

    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        Subfinder(hostname, subdomainsPath)
    }()
    go func() {
        defer wg.Done()
        Amass(hostname, subdomainsPath)
    }()
    wg.Wait()

    uniqSort(subdomainsPath)
}

func NmapScan(subdomainsPath, nmapOutPath string) {
    touch(nmapOutPath)
    fmt.Println("nmap", "-AsV", "-sC", "-iL", subdomainsPath, "-oN", nmapOutPath)
    cmd := exec.Command("nmap", "-AsV", "-sC", "-iL", subdomainsPath, "-oN", nmapOutPath)
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: babyface example.com")
        os.Exit(1)
    }
    hostname := string(os.Args[1])
    targetPath, _ := expand(fmt.Sprintf("~/.babyface/targets/%s", hostname))
    subdomainsPath := filepath.Join(targetPath, "subdomains.txt")
    nmapOutPath := filepath.Join(targetPath, "nmap.txt")

    Setup(hostname)    
    SubdomainScan(hostname, subdomainsPath)
    NmapScan(subdomainsPath, nmapOutPath)
    fmt.Println("🙃")
}