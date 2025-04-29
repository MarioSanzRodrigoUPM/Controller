package dhcpserver

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Rotchamar/dhcp/dhcpv4"
	"github.com/Rotchamar/dhcp/dhcpv4/server4"
)

var DHCPClients Clients
var EthConn EthSocketConn

// Mutex y variable global para el targetGnbID
var (
	globalTargetGnb string
	mu              sync.RWMutex
)

var (
	// IP base por GNB ID y usamos subredes completas

	subnetRanges = map[string]*net.IPNet{}
	// Para cada GNB, el siguiente offset (0,1,2,…)
	nextOffset = make(map[string]uint32)
	offsetsMu  sync.Mutex

	// La última IP calculada (puedes o no exponerla globalmente)
	assignedIP   net.IP
	assignedIPMu sync.RWMutex
)

// Inicializa las subredes
func init() {
	// Definimos aquí las subredes que queremos
	subnets := map[string]string{
		"000102": "10.2.0.0/26",  // IPs 10.2.0.2 - 10.2.0.62
		"000103": "10.2.0.64/26", // IPs 10.2.0.65 - 10.2.0.126
	}

	for gnb, cidr := range subnets {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Fatalf("Error al parsear CIDR %s para GNB %s: %v", cidr, gnb, err)
		}
		subnetRanges[gnb] = ipnet
	}
}

// DecideIP calcula y almacena assignedIP = primera IP libre válida de la subred
func DecideIP(targetGnbID string) {
	offsetsMu.Lock()
	defer offsetsMu.Unlock()

	ipnet, ok := subnetRanges[targetGnbID]
	if !ok {
		log.Printf(" No hay subred definida para GNB ID %s", targetGnbID)
		assignedIP = net.IPv4(10, 255, 255, 254)
		return
	}

	startIP := incrementIP(ipnet.IP, 2) // Saltamos .0 (red) y .1 (gateway)
	off := nextOffset[targetGnbID]

	candidate := incrementIP(startIP, off)
	if !ipInSubnet(candidate, ipnet) || isBroadcast(candidate, ipnet) {
		log.Printf("⚠️ No quedan IPs válidas disponibles en la subred para GNB ID %s", targetGnbID)
		assignedIP = net.IPv4(10, 255, 255, 254)
		return
	}

	nextOffset[targetGnbID] = off + 1

	assignedIPMu.Lock()
	assignedIP = candidate
	assignedIPMu.Unlock()

	log.Printf("[DecideIP] GNB ID: %s | Offset: %d | IP asignada: %s", targetGnbID, off, candidate)
}

// GetAssignedIP devuelve la IP calculada por la última llamada a DecideIP
func GetAssignedIP() net.IP {
	assignedIPMu.RLock()
	defer assignedIPMu.RUnlock()
	log.Printf("[GetAssignedIP] Devolviendo IP asignada: %s", assignedIP.String())
	return assignedIP
}

// incrementIP suma "n" al valor de una IP
func incrementIP(ip net.IP, n uint32) net.IP {
	ip = ip.To4()
	val := binary.BigEndian.Uint32(ip)
	val += n
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], val)
	return net.IP(buf[:])
}
func isBroadcast(ip net.IP, subnet *net.IPNet) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	mask := binary.BigEndian.Uint32(subnet.Mask)
	network := binary.BigEndian.Uint32(subnet.IP)
	broadcast := network | (^mask)
	return binary.BigEndian.Uint32(ip4) == broadcast
}

// ipInSubnet comprueba si la IP está dentro de la subred
func ipInSubnet(ip net.IP, subnet *net.IPNet) bool {
	return subnet.Contains(ip)
}

func setGlobalTargetGnb(id string) {
	mu.Lock()
	globalTargetGnb = id
	mu.Unlock()
}

func StartDHCPServer(interfaceName string) {
	var err error

	DHCPClients.Value = make(map[[6]byte]ClientInfo)

	EthConn, err = NewEthSocketConn(interfaceName)
	if err != nil {
		log.Fatal(err)
	}

	ServerIP, err = GetInterfaceIpv4Addr(interfaceName)
	if err != nil {
		log.Fatal(err)
	}

	laddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: dhcpv4.ServerPort}
	server, err := server4.NewServer(interfaceName, laddr, handlerDHCP, server4.WithDebugLogger())
	if err != nil {
		log.Fatal(err)
	}

	log.Println(" DHCP Server corriendo en UDP :67")
	server.Serve()
}

func TriggerDHCPClient(ueIP string, targerGnbID string) error {
	setGlobalTargetGnb(targerGnbID)
	DecideIP(targerGnbID) //llama a la función auxiliar para decidir la Ip en función del targetGnnID
	url := fmt.Sprintf("http://%s:8081/dhcp/start", ueIP)
	log.Printf("Lanzando GET a %s", url)

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("fallo al llamar a %s: %w", url, err)
	}
	defer resp.Body.Close()

	log.Printf(" Respuesta desde %s: %s", ueIP, resp.Status)
	return nil
}
