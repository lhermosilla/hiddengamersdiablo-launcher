import QtQuick 2.4
import QtQuick.Controls 2.5
import QtQuick.Layouts 1.3

Popup {
    id: settingsPopup

    property int itemHeight: 50
    property bool errored: false

    // Game roles describe the model id in the backend.
    // They can be found in the game model.
    property var gameRoles: { 
        "id": 257,
        "location": 258,
        "instances": 260,
        "override_bh_cfg": 264,
        "flags": 272,
        "hd_version": 288,
        "maphack_version": 320
    }

    modal: true
    focus: true
    width: 850
    height: 520
    margins: 0
    padding: 0
    
    anchors.centerIn: root
    closePolicy: Popup.NoAutoClose

    Overlay.modal: Item {
        Rectangle {
            anchors.fill: parent
            color: "#000000"
            opacity: 0.8
        }
    }

    Rectangle {
        color: "#0f0f0f"
        border.color: "#000000"
        border.width: 1
        anchors.fill: parent

        // Bottom background.
        Image {
            width: 848
            source: "assets/stone_bg.png";
            fillMode: Image.PreserveAspectFit;
            anchors.bottom: parent.bottom
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.bottomMargin: 1
        }

        RowLayout {
            id: settingsLayout
            anchors.fill: parent
            spacing: 8

             // Left column.
            Item {
                visible: (gamesList.count > 0)
                Layout.fillWidth: true
                Layout.minimumWidth: 300
                Layout.preferredWidth: 300
                Layout.maximumWidth: 300
                Layout.fillHeight: true

                Title {
                    text: "AJUSTES DEL JUEGO"
                    anchors.top: parent.top
                    anchors.left: parent.left
                    anchors.topMargin: 20
                    font.pixelSize: 15
                    font.bold: true
                    leftPadding: 30
                }

                ListView {
                    id: gamesList
                    width: parent.width - 30;
                    height: gamesList.count * itemHeight
                    anchors.top: parent.top
                    anchors.right: parent.right
                    anchors.topMargin: 50

                    // Disable scroll.
			        interactive: false

                    model: settings.games
                    delegate: SettingsDelegate{}

                    onCurrentItemChanged: {   
                        if(gamesList.count > 0) {
                            updateGame()
                        }
                    }
                }

                // Add new game button.
                Title {
                    visible: (gamesList.count <= 3)
                    text: "+ Agregar instalacion Diablo II"
                    anchors.top: gamesList.bottom
                    anchors.left: parent.left
                    anchors.topMargin: 20
                    font.bold: true
                    leftPadding: 30

                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onClicked: {
                            // Add the game to the model, without persisting it to the store.
                            settings.addGame()

                            // Set last index as current.
                            gamesList.currentIndex = (gamesList.count-1)
                        }
                    }
                }
            }

             // Right column.
            Item {
                Layout.fillWidth: true
                Layout.fillHeight: true

                // Visible if there are no games set up.
                Item {
                    id: intro
                    visible: (gamesList.count == 0)
                    width: 620
                    height: (parent.height * 0.80)
                    anchors.centerIn: parent

                    Column {
                        spacing: 2

                        Item {
                            height: 44
                            width: intro.width

                            Title {
                               text: "BIENVENIDO AL LAUNCHER DE HIDDENGAMERS DIABLO"
                               font.pixelSize: 20
                            }

                            Separator{}
                        }

                        Item {
                            height: 60
                            width: intro.width

                            SText {
                                text: "Antes de poder jugar, debe configurar las ubicaciones de sus juegos. Puede configurar varios directorios de juegos con diferentes configuraciones, como HD mod o maphack."
                                width: parent.width
                                anchors.top: parent.top
                                anchors.left: parent.left
                                anchors.topMargin: 10
                                font.pixelSize: 12
                                color: "#a3a3a3"
                                wrapMode: Text.WordWrap 
                            }
                        }

                        Item {
                            height: 30
                            width: intro.width

                            Title {
                               text: "COMO FUNCIONA"
                               font.pixelSize: 16
                            }

                            Separator{}
                        }

                        Item {
                            height: 40
                            width: intro.width

                            Row {
                                height: parent.height
                                spacing: 10

                                Rectangle {
                                    color: "#3b0000"
                                    width: 24
                                    height: 24
                                    radius: 12
                                    anchors.verticalCenter: parent.verticalCenter

                                    Title {
                                        text: "1"
                                        color: "#fff"
                                        anchors.centerIn: parent
                                    }
                                }

                                SText {
                                    text: "Configure uno o varios juegos de Diablo II que haya instalado"
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: "#a3a3a3"
                                }
                                
                            }

                            Separator{}
                        }

                        Item {
                            height: 40
                            width: intro.width

                            Row {
                                height: parent.height
                                spacing: 10

                                Rectangle {
                                    color: "#3b0000"
                                    width: 24
                                    height: 24
                                    radius: 12
                                    anchors.verticalCenter: parent.verticalCenter

                                    Title {
                                        text: "2"
                                        color: "#fff"
                                        anchors.centerIn: parent
                                    }
                                }

                                SText {
                                    text: "Elija cuántas instancias lanzar y si desea incluir HD o maphack"
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: "#a3a3a3"
                                }
                                
                            }

                            Separator{}
                        }

                        Item {
                            height: 40
                            width: intro.width

                            Row {
                                height: parent.height
                                spacing: 10

                                Rectangle {
                                    color: "#3b0000"
                                    width: 24
                                    height: 24
                                    radius: 12
                                    anchors.verticalCenter: parent.verticalCenter

                                    Title {
                                        text: "3"
                                        color: "#fff"
                                        anchors.centerIn: parent
                                    }
                                }

                                SText {
                                    text: "El lanzador determinará si necesitas parchear los juegos para estar al día con ManaosDiablo"
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: "#a3a3a3"
                                }
                                
                            }

                            Separator{}
                        }

                        Item {
                            height: 40
                            width: intro.width

                            Row {
                                height: parent.height
                                spacing: 10

                                Rectangle {
                                    color: "#3b0000"
                                    width: 24
                                    height: 24
                                    radius: 12
                                    anchors.verticalCenter: parent.verticalCenter

                                    Title {
                                        text: "4"
                                        color: "#fff"
                                        anchors.centerIn: parent
                                    }
                                }

                                SText {
                                    text: "Una vez finalizado el parche, estará listo para jugar."
                                    anchors.verticalCenter: parent.verticalCenter
                                    color: "#a3a3a3"
                                } 
                            }

                            Separator{}
                        }

                        Item {
                            height: 80
                            width: intro.width

                            PlainButton {
                                width: 200
                                height: 50
                                label: "EMPEZAR"
                                anchors.top: parent.top
                                anchors.topMargin: 15

                                onClicked: settings.addGame()
                            }
                        }
                    }
                }

                 // Settings shown if there are games already setup    
                Item {
                    visible: (gamesList.count > 0)
                    anchors.fill: parent

                    GameSettings {
                        id: gameSettings
                        anchors.left: parent.left
                        anchors.top: parent.top
                        anchors.topMargin: 40
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                }

                // Error popup.
                Item {
                    id: errorPopup
                    visible: errored
                    width: 300
                    height: 50
                    anchors.horizontalCenter: doneButton.horizontalCenter
                    anchors.bottom: doneButton.top
                    anchors.bottomMargin: 15

                    Rectangle {
                        anchors.fill: parent
                        color: "#8f3131"
                        border.width: 1
                        border.color: "#000000"
                    }


                    SText {
                        text: "El juego nuevo no tiene configurado el directorio de Diablo II."
                        font.pixelSize: 11
                        anchors.centerIn: parent
                        color: "#ffffff"
                    }
                }
                
                PlainButton {
                    id: doneButton
                    visible: (gamesList.count > 0)
                    label: "LISTO"
                    width: 100
                    height: 50
                    anchors.bottom: parent.bottom
                    anchors.left: parent.left
                    anchors.bottomMargin: -20
                    anchors.leftMargin: 65

                    onClicked: {
                        // Reset error.
                        errored = false
                        
                        if(validateGames()) {
                            var success = settings.persistGameModel()
                            if(success) {
                                // Validate the game versions after changes has been made to the settings.
                                diablo.patchFiles.clear()
                                diablo.validateVersion()
                                settingsPopup.close()
                            }
                        } else {
                            // Show error.
                            errored = true

                            // Remove error after a timeout.
                            errorTimer.restart()
                        }
                    }
                }
            }
        }
    }

    // validateGames will validate that the input is correctly set.
    function validateGames() {
       for(var i = 0; i < gamesList.count; i++) {
            var location = gamesList.model.data(gamesList.model.index(i, 0), gameRoles.location)
            if(location.length == 0) {
                return false
            }
        }
        
        return true
    }

    function updateGame() {
        var model = settings.games

        // Only update if any games exist.
        if(gamesList.currentIndex != -1) {
            gameSettings.setGame({
                "id": model.data(model.index(gamesList.currentIndex, 0), gameRoles.id),
                "location": model.data(model.index(gamesList.currentIndex, 0), gameRoles.location),
                "instances": model.data(model.index(gamesList.currentIndex, 0), gameRoles.instances),
                "maphack": model.data(model.index(gamesList.currentIndex, 0), gameRoles.maphack),
                "override_bh_cfg": model.data(model.index(gamesList.currentIndex, 0), gameRoles.override_bh_cfg),
                "hd": model.data(model.index(gamesList.currentIndex, 0), gameRoles.hd),
                "flags": model.data(model.index(gamesList.currentIndex, 0), gameRoles.flags),
                "hd_version": model.data(model.index(gamesList.currentIndex, 0), gameRoles.hd_version),
                "maphack_version": model.data(model.index(gamesList.currentIndex, 0), gameRoles.maphack_version),
            })
        }
    }

    Timer {
        id: errorTimer
        interval: 5000; running: false; repeat: false
        onTriggered: errored = false
    }

    onAboutToShow: {
        updateGame()
    }
}
