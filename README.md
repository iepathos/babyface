# babyface

babyface is a small wrapper around subfinder, massdns, and nmap.  Given a hostname, it finds subdomains and then performs a noisy nmap check on the subdomains hostname.

this tool may be expanded to filter out the boring targets from the nmap scan and present the interesting ones.