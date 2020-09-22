import QtQuick 2.12
import QtQuick.Controls 1.4
import QtQuick.Controls.Styles 1.4

Item {
    id: patcher
    property bool patchFilesHovered: false
    height: 120
    anchors.left: parent.left
    anchors.leftMargin: 20
    anchors.verticalCenter: parent.verticalCenter

    Item {
        anchors.fill: parent
        visible: diablo.validatingVersion

        // Loading circle.			
        CircularProgress {
            size: 20
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            visible: diablo.validatingVersion
        }

        Title {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            anchors.leftMargin: 35
            text: "Comprobando version del juego..."
            font.pixelSize: 15
        }
    }

    // Show when we're patching and no error has occurred.
    Item {
        anchors.fill: parent 
        visible: (diablo.patching && !diablo.errored && !diablo.validatingVersion)

        ProgressBar {
            height: 8
            value: diablo.patchProgress 
            width: parent.width
            anchors.verticalCenter: parent.verticalCenter
            
            style: ProgressBarStyle {
                background: Rectangle {
                    radius: 3
                    color: "#1e1b26"
                    border.width: 1
                    border.color: "#000000"
                    opacity: 0.6
                }
                
                progress: Rectangle {
                    radius: 3
                    color: "#5c0202"
                    border.width: 1
                    border.color: "#000000"
                }
            }
        }

        SText {
            anchors.bottom: parent.bottom;
            anchors.bottomMargin: 40
            font.family: beaufortbold.name
            text: diablo.status
            font.pixelSize: 12
        }
    }

    // Show when patcher errors.
    Item {
        anchors.fill:parent 
        visible: diablo.errored && !diablo.validatingVersion

        Title {
            id: patchError
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            anchors.leftMargin: 20
            text: "No se pueden encontrar los archivos"
            font.pixelSize: 15
            color: "#8f3131"
        }

        PlainButton {
            width: 120
            height: 40
            label: "INTENTE NUEVAMENTE"
            fontSize: 12
            anchors.verticalCenter: parent.verticalCenter
            anchors.left: patchError.right
            anchors.leftMargin: 20

            onClicked: {
                diablo.applyPatches()
            }
        }
    }

    // Show when patching is done, no error occurred and the game version is valid.
    Item {
        anchors.fill:parent 
        visible: (!diablo.patching && !diablo.errored && !diablo.validatingVersion && diablo.validVersion)

        Title {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            anchors.leftMargin: 30
            text: "Juego actualizado a la fecha"
            font.pixelSize: 15
        }

        Item {
            width: 300; height: parent.height
            anchors.verticalCenter: parent.verticalCenter
            anchors.right: parent.right;

            Title {
                anchors.right: launchDelay.left
                anchors.top: parent.top
                anchors.topMargin: 6
                anchors.rightMargin: 5
                text: "Retardar lanzamiento"
                font.pixelSize: 12
             }

            Dropdown{
                id: launchDelay
                anchors.bottom: playButton.top
                anchors.bottomMargin: 5
                anchors.rightMargin: 13
                anchors.right: parent.right
                currentIndex: 0
                model: ["1 seg", "2 seg", "3 seg", "4 seg", "5 seg"]
                height: 30
                width: 70

                // Sets the correct index when the component has loaded.
                Component.onCompleted: {
                    // If launch delay hasn't been set, set index 0.
                    if(diablo.launchDelay == 0) {
                        this.currentIndex = 0
                        return
                    }

                    this.currentIndex = (diablo.launchDelay / 1000)-1
                }

                onActivated: {
                    var delay = 1000
                    switch(this.currentText) {
                        case "1 seg":
                            delay = 1000
                            break;
                        case "2 seg":
                            delay = 2000
                            break;
                        case "3 seg":
                            delay = 3000
                            break;
                        case "4 seg":
                            delay = 4000
                            break;
                        case "5 seg":
                            delay = 5000
                            break;

                    }
                    diablo.updateLaunchDelay(delay)
                }
            }

            // Launch button.
            PlainButton {
                id: playButton
                label: (diablo.launching ? "INICIANDO..." : "JUGAR")
                fontSize: 15
                clickable: (!diablo.launching)
                width: 275; height: 50
                backgroundColor: "#3b0000"
                colorHovered: "#5c0202"
                anchors.verticalCenter: parent.verticalCenter
                anchors.horizontalCenter: parent.horizontalCenter

                onClicked: diablo.launchGame()
            }
        }
    }

    // Show when the Diablo version is invalid, we're not patching and there's no error.
    Item {
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        width: 400
        height: 40
        visible: (!diablo.validVersion && !diablo.patching && !diablo.errored && !diablo.validatingVersion)

        Title {
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            anchors.leftMargin: 20
            text: "El juego necesita actualizarse"
            font.pixelSize: 15
        }

        PlainButton {
            width: 120
            height: 40
            label: "ACTUALIZAR"
            fontSize: 12
            anchors.top: parent.top
            anchors.right: patchChanges.left
            anchors.rightMargin: 10

            onClicked: {
                // Clear any files that has been set to be patched before patching.
                diablo.patchFiles.clear()
                
                diablo.applyPatches()
            }
        }

        Image {
            id: patchChanges
            fillMode: Image.Pad
            anchors.verticalCenter: parent.verticalCenter
            anchors.right: parent.right
            width: 16
            height: 16
            source: "assets/icons/patch.png"
            opacity: patchFilesHovered ? 1.0 : 0.5

            MouseArea {
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: patchPopup.open()

                onEntered: {
                    patchFilesHovered = true
                }
                onExited: {
                    patchFilesHovered = false
                }
            }
        }
    }

    Component.onCompleted: {
        // If any games have been set, check their versions.
        if(settings.games.rowCount() > 0) {
            // Clear any files that needs to be patched before revalidating.
            diablo.patchFiles.clear()
            
            diablo.validateVersion()
        }
    }
}
