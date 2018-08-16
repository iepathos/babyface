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

func Setup(hostname string) {
    // setup ~/.babyface
    path, _ := expand("~/.babyface")
    targetsPath := filepath.Join(path, "targets")
    targetHostPath := filepath.Join(targetsPath, hostname)
    if _, err := os.Stat(path); os.IsNotExist(err) {
        _ = os.Mkdir(path, os.ModePerm)
    }
    if _, err := os.Stat(targetsPath); os.IsNotExist(err) {
        _ = os.Mkdir(targetsPath, os.ModePerm)
    }
    if _, err := os.Stat(targetHostPath); os.IsNotExist(err) {
        _ = os.Mkdir(targetHostPath, os.ModePerm)
    }
}

func subfinder(hostname, subdomainsPath string) {
    fmt.Println("subfinder -d", hostname, "-o", subdomainsPath)
    cmd := exec.Command("subfinder", "-d", hostname, "-o", subdomainsPath)
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func amass(hostname, subdomainsPath string) {
    fmt.Println("amass -d", hostname, "-o", subdomainsPath, "-brute")
    cmd := exec.Command("amass", "-d", hostname, "-o", subdomainsPath, "-brute")
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
}

func readLines(file string) (lines []string, err error) {
    f, err := os.Open(file)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    r := bufio.NewReader(f)
    for {
        const delim = '\n'
        line, err := r.ReadString(delim)
        if err == nil || len(line) > 0 {
            if err != nil {
                line += string(delim)
            }
            lines = append(lines, line)
        }
        if err != nil {
            return nil, err
        }
    }
    return lines, nil
}

func writeLines(file string, lines []string) (err error) {
    f, err := os.Create(file)
    if err != nil {
        return err
    }
    defer f.Close()
    w := bufio.NewWriter(f)
    defer w.Flush()
    for _, line := range lines {
        _, err := w.WriteString(line)
        if err != nil {
            return err
        }
    }
    return nil
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

func SubdomainScan(hostname, subdomainsPath string) {
    touch(subdomainsPath)

    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        subfinder(hostname, subdomainsPath)
    }()
    go func() {
        defer wg.Done()
        amass(hostname, subdomainsPath)
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
}