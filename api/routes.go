package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	dhcp "github.com/krepox/Controller/dhcpserver"
)

type AgfId struct {
	GnbID string `json:"gnbId" binding:"required"`
}
type User struct {
	Supi  string `json:"supi"  binding:"required"`
	IMSI  string `json:"imsi" binding:"required"`
	GnbID string `json:"gnb_id" binding:"required"`
}

var agfIds []AgfId

// var users []User
var users = make([]User, 0)         // slice para listar
var userMap = make(map[string]User) // key = supi

// RegisterRoutes agrega las rutas al router principal
func RegisterRoutes(router *gin.Engine) {
	router.GET("/agfs", getAgfs)
	router.POST("/AGF_registration", registerAgf)
	router.POST("/triggerDHCP", triggerDHCP)
	router.POST("/user_registration", registerUser)
	router.GET("/users", getUsers)
	router.POST("/triggerHandover", triggerHandover) //endpoint para iniciar el H0 en un AGF
}

func getAgfs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"agfs": agfIds})
}

func registerAgf(c *gin.Context) {
	var d AgfId
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agfIds = append(agfIds, d)
	c.JSON(http.StatusOK, gin.H{
		"message": "Datos almacenados correctamente",
		"gnbId":   d.GnbID,
	})
}

/*
	func registerUser(c *gin.Context) {
		var u User
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		users = append(users, u)
		c.JSON(http.StatusOK, gin.H{
			"message": "Usuario registrado correctamente",
			"imsi":    u.IMSI,
			"gnb_id":  u.GnbID,
		})
	}
*/
func registerUser(c *gin.Context) {
	var u User
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	users = append(users, u)
	userMap[u.Supi] = u
	c.JSON(http.StatusOK, gin.H{"message": "Usuario registrado", "supi": u.Supi})
}

func getUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// se modifica la función ya que ahora es un POST
// necesitamos saber cual es la Ip del usuario para enviarla como param, en vez de ponerla a mano con el curl
// el AGF tiene la Ip guardada en el teidupf
// hay que buscar una forma de que el controlador sepa las Ips de los usuarios, ya que, cuando se registran los usuasrios se hace por imsi y supi (forma de identificarlos)
func triggerDHCP(c *gin.Context) {
	//aqui guardar el target gnbid para dependiendo ver a que ip moverlo
	// Esperamos un JSON con el campo "ue" (IP del usuario)
	var payload struct {
		UeIP        string `json:"ue"`
		TargetGnbID string `json:"gnbId"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil || payload.UeIP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Falta parámetro 'ue'"})
		return
	}

	if err := dhcp.TriggerDHCPClient(payload.UeIP, payload.TargetGnbID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente DHCP activado correctamente"})
}

// triggerHandover activa el Handover del AGF
/* func triggerHandover(c *gin.Context) {
	var gnbPayload struct {
		GnbId string `json:"gnbId"`
	}

	// Obtener el GNB ID del cuerpo de la solicitud
	if err := c.ShouldBindJSON(&gnbPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Determinar la IP del AGF basándonos en el GNB ID
	var agfIP string
	switch gnbPayload.GnbId {
	case "000102":
		agfIP = "10.2.0.80"
	case "000103":
		agfIP = "10.2.0.85"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "GnbId no reconocido"})
		return
	}
	// Llamar al servidor AGF enviando una solicitud POST
	url := fmt.Sprintf("http://%s:8082/triggerHandover", agfIP) // Usamos la IP dinámica del AGF

	// Serializar el JSON para enviar al servidor AGF
	jsonData, err := json.Marshal(gnbPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Crear una solicitud HTTP POST para el AGF
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error al hacer la solicitud POST de inicio de handover: %v", err)
		return
	}
	defer resp.Body.Close()
	// Verificar respuesta
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error en respuesta al iniciar el handover %s", resp.Status)
	} else {
		log.Println("Handover iniciado exitosamente")
	}
	// Responder al cliente que la solicitud de Handover fue procesada
	c.JSON(http.StatusOK, gin.H{
		"message": "Handover activado correctamente en AGF",
	})
}
*/
// triggerHandover activa el Handover del AGF
func triggerHandover(c *gin.Context) {
	// 1) Estructura que esperamos del frontend
	type HandoverReq struct {
		Supi  string `json:"supi"  binding:"required"`
		GnbId string `json:"gnbId" binding:"required"` // este es el destino
	}

	var req HandoverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Petición de Handover recibida - SUPI: %s, GNB destino: %s", req.Supi, req.GnbId)

	// 2) Buscar el usuario en el mapa para saber en qué AGF está conectado ahora
	user, ok := userMap[req.Supi]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	log.Printf("Usuario SUPI %s registrado en AGF origen (GNB ID): %s", user.Supi, user.GnbID)

	// 3) Resolver IP del AGF origen (donde está el UE actualmente)
	var agfIP string
	switch user.GnbID {
	case "000102":
		agfIP = "10.2.0.80"
	case "000103":
		agfIP = "10.2.0.85"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "GnbID origen no reconocido"})
		return
	}

	log.Printf("Enviando solicitud de HO al AGF origen con IP: %s", agfIP)

	// 4) Enviar la estructura original (supi + gnb destino) al AGF origen
	url := fmt.Sprintf("http://%s:8082/triggerHandover", agfIP)
	jsonData, _ := json.Marshal(req)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error al hacer POST al AGF origen: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "AGF inalcanzable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error en respuesta del AGF: %s", resp.Status)
	} else {
		log.Println("Handover iniciado exitosamente en AGF origen")
		// ACTUALIZAMOS el campo GnbID en memoria tras el HO exitoso
		user.GnbID = req.GnbId
		userMap[req.Supi] = user
		for i, u := range users {
			if u.Supi == req.Supi {
				users[i].GnbID = req.GnbId
				break
			}
		}
		log.Printf("SUPI %s movido exitosamente al AGF %s", req.Supi, req.GnbId)

	}

	// 5) Respuesta al frontend
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Handover solicitado: SUPI %s ➜ GNB %s", req.Supi, req.GnbId),
	})
}
