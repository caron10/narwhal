package main

import "fmt"
import "os"
import "os/signal"
import "syscall"
import "os/exec"
import "io/ioutil"
import "time"
import "strings"
import "strconv"

func main() {
    fmt.Println("started")

    configFile := os.Args[1]
    fmt.Println("config file: " + configFile)
    pidFile := os.Args[2]
    fmt.Println("pid file: " + pidFile)

    sigs := make(chan os.Signal, 1)
    done := make(chan bool, 1)
    signal.Notify(sigs, syscall.SIGTERM)

    check_and_run(configFile, pidFile)

    ticker := time.NewTicker(5 * time.Second)

    // This goroutine executes a blocking receive for
    // signals. When it gets one it'll restart unicorn
    // and then notify the program that it can finish.
    go func() {
        sig := <-sigs
        fmt.Println("received signal")
        fmt.Println(sig)
        ticker.Stop()
        restart_unicorn(get_unicorn_pid(pidFile))
        fmt.Println("new unicorn pid " + get_unicorn_pid(pidFile))
        done <- true
    }()

    go func() {
        for time := range ticker.C {
            fmt.Println("checking unicorn at", time)
            check_and_run(configFile, pidFile)
        }
    }()

    fmt.Println("awaiting signal")
    <-done
    fmt.Println("exiting")
}

func start_unicorn(configFile string) {
    fmt.Println("starting unicorn")
    cmd := exec.Command("unicorn", "-c", configFile, "-D")
    out, err := cmd.CombinedOutput()
    fmt.Println(string(out))
    if err != nil {
        panic(err)
    }
}

func restart_unicorn(pid string) {
    fmt.Println("restarting unicorn with pid " + pid)

    intPid, err := strconv.Atoi(pid)
    if err != nil {
        panic(err)
    }

    process, err := os.FindProcess(intPid)
    if err != nil {
        panic(err)
    }

    process.Signal(syscall.SIGUSR2)
    time.Sleep(5 * time.Second)
    process.Signal(syscall.SIGQUIT)
}

func get_unicorn_pid(pidFile string) string {
    fmt.Println("detecting unicorn pid")
    dat, err := ioutil.ReadFile(pidFile)
    if err != nil {
        fmt.Println(err)
        return ""
    }

    pid := strings.TrimSpace(string(dat))
    fmt.Println(pid)
    return pid
}

func check_unicorn_running(pidFile string) bool {
    pid := get_unicorn_pid(pidFile)
    if pid == "" {
        fmt.Println("no pid file")
        return false
    }

    intPid, err := strconv.Atoi(pid)
    if err != nil {
        panic(err)
    }

    process, err := os.FindProcess(intPid)
    if err != nil {
        fmt.Println(err)
        return false
    }

    err = process.Signal(syscall.Signal(0))
    if err != nil {
        fmt.Println("no process found")
        return false
    }

    return true
}

func check_and_run(configFile string, pidFile string) {
    if !check_unicorn_running(pidFile) {
        fmt.Println("no unicorn is running")
        start_unicorn(configFile)
    } else {
        fmt.Println("unicorn is running")
    }
}
