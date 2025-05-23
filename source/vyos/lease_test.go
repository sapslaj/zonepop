package vyos

import "testing"

func TestLeasesFromJSON(t *testing.T) {
	t.Parallel()

	input := `
[
  {
    "start": "2023/02/27 04:19:32",
    "end": "2023/02/28 04:19:32",
    "remaining": "22:48:00",
    "tstp": "",
    "tsfp": "",
    "atsfp": "",
    "cltt": "1 2023/02/27 04:19:32",
    "hardware_address": "52:54:00:d2:12:06",
    "hostname": "maki",
    "state": "active",
    "ip": "172.24.5.199",
    "pool": "LAN_Internal"
  },
  {
    "start": "2023/02/27 02:59:16",
    "end": "2023/02/28 02:59:16",
    "remaining": "21:27:44",
    "tstp": "",
    "tsfp": "",
    "atsfp": "",
    "cltt": "1 2023/02/27 02:59:16",
    "hardware_address": "52:54:00:9e:5e:ec",
    "hostname": "zerotwo",
    "state": "active",
    "ip": "172.24.4.200",
    "pool": "LAN_Servers"
  }
]
	`

	leases, err := LeasesFromJSON([]byte(input))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("len(leases) == %d ; expected 2", len(leases))
	}
	lease := leases[0]
	if lease.Pool != "LAN_Internal" {
		t.Fatalf("lease.Pool == %s ; expected `LAN_Internal`", lease.Pool)
	}
	if lease.IP != "172.24.5.199" {
		t.Fatalf("lease.IP == %s ; expected `172.24.5.199`", lease.IP)
	}
	if lease.Hostname != "maki" {
		t.Fatalf("lease.Hostname == %s ; expected `maki`", lease.Pool)
	}
	if lease.HardwareAddress != "52:54:00:d2:12:06" {
		t.Fatalf("lease.HardwareAddress == %s ; expected `52:54:00:d2:12:06`", lease.HardwareAddress)
	}
}

func TestLeasesFromShowOutput(t *testing.T) {
	t.Parallel()

	input := `
IP Address    MAC address        State    Lease start          Lease expiration     Remaining    Pool            Hostname                   Origin
------------  -----------------  -------  -------------------  -------------------  -----------  --------------  -------------------------  --------
172.24.5.199  52:54:00:d2:12:06  active   2023/02/27 04:19:32  2023/02/28 04:19:32  22:48:00     LAN_Internal    maki                       local
172.24.4.200  52:54:00:9e:5e:ec  active   2023/02/27 02:59:16  2023/02/28 02:59:16  21:27:44     LAN_Servers     zerotwo                    local
`

	leases, err := LeasesFromShowOutput([]byte(input))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("len(leases) == %d ; expected 2", len(leases))
	}
	lease := leases[0]
	if lease.Pool != "LAN_Internal" {
		t.Fatalf("lease.Pool == %s ; expected `LAN_Internal`", lease.Pool)
	}
	if lease.IP != "172.24.5.199" {
		t.Fatalf("lease.IP == %s ; expected `172.24.5.199`", lease.IP)
	}
	if lease.Hostname != "maki" {
		t.Fatalf("lease.Hostname == %s ; expected `maki`", lease.Pool)
	}
	if lease.HardwareAddress != "52:54:00:d2:12:06" {
		t.Fatalf("lease.HardwareAddress == %s ; expected `52:54:00:d2:12:06`", lease.HardwareAddress)
	}
}
