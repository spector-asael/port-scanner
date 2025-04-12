# PRESENTATION VIDEO: https://youtu.be/EKrqY5OCytA?si=Pa-uhXNdvd0RP5KI

# This is is port scanner written in Go. It allows users to scan specified target hosts for open ports. The tool uses concurrency to speed up scanning and provides JSON output for easier parsing.

# Scan hosts using -target, or -targets (comma-separated list) for multiple hosts.

# Specify port range with -start-port and -end-port or provide a list with -ports (comma-separated list).

# Control concurrency with -workers to optimize performance.

# Set timeouts with -timeout for better control over slow responses.

# Use -json to have your summary outputted in JSON format

# To build the program, use:
# go build -o portscanner main.go
# To run the program, use ./portscanner along with the flags you wish to use.

# Example usage:
# ./portscanner -target=scanme.nmap.org -start-port=1 -end-port=500 -workers 100
# ./portscanner -target=scanme.nmap.org -ports=22,80,443,8080 -json 

# Example output:
# Scanning port 500/500...
# Report summary.
# Time elapsed: 17.66s
# Total number of ports scanned: 500
# Open ports found: [ scanme.nmap.org:22 scanme.nmap.org:80 ]

#  "open_ports": [
#    {
#      "target": "scanme.nmap.org",
#      "port": 22,
#      "status": "open",
#      "banner": "SSH-2.0-OpenSSH_6.6.1p1 Ubuntu-2ubuntu2.13"
#    },
#    {
#      "target": "scanme.nmap.org",
#      "port": 80,
#      "status": "open"
#    }
#  ] 
# }
