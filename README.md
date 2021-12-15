# HiddenGamersDiablo launcher

![launcher imagen](/docs/launcher.png)

## Sobre el proyecto

El lanzador HiddenGamersDiablo es un launcher de juegos multiplataforma para Diablo II y específicamente para la comunidad [HiddenGamersDiablo] (https://hiddengamers.cl/). Fue creado para ayudar a los nuevos jugadores a instalar parches, actualizar registros y ayudar con otros problemas técnicos para reducir la barrera de entrada a la comunidad de HiddenGamersDiablo, al mismo tiempo que ayuda a los jugadores más experimentados con configuraciones más avanzadas, como mods HD y el lanzamiento de múltiples cajas.

## Features

- [x] Parchea cualquier* Diablo II LOD a la versión 1.14b
- [x] Aplica el parche de HiddenGamersDiablo automáticamente
- [x] Parche una lista de acciones - sabe exactamente que archivos actualizar
- [x] Permite múltiples instalaciones de Diablo II con diferentes ajustes (como el Maphack & HD)
- [x] Instala automáticamente y actualia el Maphack & mod HD
- [x] Ejecuta múltiples Diablo II desde múltiples instalaciones
- [x] Resuelve el problema de Access Violation (DEP)
- [x] Funciona con el Glide Wrapper
- [x] Soporta muchos parametros de lanzamiento populares

### Soporte completo Sistemas Operativos

- [x] Windows
- [ ] OSX (faltan algunas características específicas de D2)
- [ ] Linux (faltan algunas características específicas de D2)

## Desarrollo

### Go

Instalá Go 1.12 o superior siguiendo las [instrucciones de instalación](http://golang.org/doc/install.html) para su sistema operativo.

### Bindings Qt para Go

Antes de poder compilar, debe instalar los [enlaces Go / Qt](https://github.com/therecipe/qt/wiki/Installation#regular-installation).

### Instale Qt5

#### OSX

En OSX, usar brew es, con mucho, la forma más sencilla de instalar Qt5.

```bash
$ brew install qt
```

#### Windows

Utilice el [instalador](https://download.qt.io/official_releases/qt/5.13/5.13.0/qt-opensource-windows-x86-5.13.0.exe) proporcionado por Qt (asegúrese de instalar el MinGW de Qt).

#### Construyendo el lanzador de HiddenGamersDiablo

```bash
# Obtener fuente de enlace
$ go get -u -v -tags=no_env github.com/therecipe/qt/cmd/...

# Descarga el repositorio con dependencias
$ go get -d -u -v github.com/lhermosilla/hiddengamersdiablo-launcher

# Construye el lanzador
$ cd $(go env GOPATH)/src/github.com/lhermosilla/hiddengamersdiablo-launcher
$ qtdeploy build

# Iniciar lanzador (diferente según el sistema operativo)
$ ./deploy/darwin/hiddengamersdiablo-launcher.app/Contents/MacOS/hiddengamersdiablo-launcher
```

## Deploying

La implementación en un objetivo se puede realizar desde cualquier sistema operativo host si hay una imagen de docker disponible; de lo contrario, el sistema operativo objetivo y el host deben ser iguales.

### Windows

#### Construyendo en docker

```bash
$ docker pull therecipe/qt:windows_64_static
$ qtdeploy -docker build windows_64_static
```

#### Construyendo en maquina local

```bash
$ qtdeploy build desktop
```

#### Actualización de la versión binaria de la aplicación y los datos del manifest

```bash
# Descargar la herramienta Governsioninfo
$ go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo

# Realice sus cambios en el archivo de manifiesto.
$ vim versioninfo.json

# Genere un nuevo resource.syso que incluya el manifiesto.
$  go generate
```

### MacOS (solo desde MacOS)

```bash
$ qtdeploy build darwin github.com/lhermosilla/hiddengamersdiablo-launcher
```

### Creditos

Agradezco eternamente al creador de este gran proyecto: @Nokka
