### Description
this package filter all tcp packages from specific local port.

main function is get latency between stratum and miners (they use websocket protocol).
measure based on fact that tcp connections send ack after receiving data.

sample output for 3 socket connections to one server:
```bash
# connection A request and response
2024-05-08 13:50:09.025572 lo    In  IP localhost.http-alt > localhost.50484: Flags [P.], seq 462:616, ack 1, win 260, options [nop,nop,TS val 4198719814 ecr 4198716815], length 154: HTTP
2024-05-08 13:50:09.025585 lo    In  IP localhost.50484 > localhost.http-alt: Flags [.], ack 616, win 258, options [nop,nop,TS val 4198719814 ecr 4198719814], length 0

# connection B request and response
2024-05-08 13:50:09.027741 lo    In  IP localhost.http-alt > localhost.50486: Flags [P.], seq 462:616, ack 1, win 260, options [nop,nop,TS val 4198719816 ecr 4198716817], length 154: HTTP
2024-05-08 13:50:09.027747 lo    In  IP localhost.50486 > localhost.http-alt: Flags [.], ack 616, win 258, options [nop,nop,TS val 4198719816 ecr 4198719816], length 0

# connection C request and response
2024-05-08 13:50:09.031903 lo    In  IP localhost.http-alt > localhost.50502: Flags [P.], seq 462:616, ack 1, win 260, options [nop,nop,TS val 4198719820 ecr 4198716820], length 154: HTTP
2024-05-08 13:50:09.031909 lo    In  IP localhost.50502 > localhost.http-alt: Flags [.], ack 616, win 258, options [nop,nop,TS val 4198719820 ecr 4198719820], length 0
```

### Dependencies
* tcpdump
```bash
sudo apt install tcpdump
yum install tcpdump
```
* sudoers patch
```bash
which tcpdump
# add this line to /etc/sudoers
alejandro ALL = NOPASSWD: /usr/bin/tcpdump
```

observe connections 
```bash
sudo tcpdump -i any -tttt 'tcp port 3333 and (tcp[tcpflags] & (tcp-push|tcp-ack) != 0)'
```

### Usage
