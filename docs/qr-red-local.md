# QR del boleto en otra red Wi-Fi

Inicia el proyecto desde PowerShell con:

```powershell
.\scripts\start-project.ps1
```

El script detecta la IPv4 de la conexión Wi-Fi o Ethernet activa, levanta Docker y mantiene actualizada la dirección del QR mientras el proyecto está abierto. En este equipo la dirección se guarda en `backend/data/qr-lan-url.txt`; el backend la lee cuando genera un QR, sin guardar la IP anterior en el boleto.

Cuando cambies de red, espera unos segundos y **genera o descarga nuevamente el QR, PDF o pase**. Un QR impreso o un pase descargado anteriormente conserva la URL que tenía al generarse. El QR del boleto abre `/verificar/{id}` en el frontend y esa página comprueba el token con la API. El QR para descargar el pase abre `/pase/{id}/billetera`.

El teléfono y la computadora deben poder comunicarse. Antes de probar el QR, abre en el teléfono la dirección `http://IP_ACTUAL:3001` que muestra la aplicación junto al QR. Si no abre, revisa el firewall de Windows y si la red de la universidad permite conexiones entre dispositivos; algunas redes aíslan a sus usuarios y un cambio de IP en el QR no puede superar ese bloqueo. En ese caso usa una red que sí permita acceso local, como un punto de acceso propio, o publica el proyecto mediante una URL accesible desde Internet.
