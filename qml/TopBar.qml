import QtQuick 2.12
import QtQuick.Layouts 1.3

Item {
    id: topbar
    property string activeMenuItem: "launch"
    property bool settingsHovered: false
    property var menuSources: { 
        "launch": "LauncherView.qml",
        "ladder": "LadderView.qml",
        "community": "CommunityView.qml"
    }

    // Main menu.
    Item {
        width: 330
        height: parent.height
        anchors.left: parent.left
        anchors.leftMargin: 20
        
        RowLayout {
            id: mainMenu
            anchors.fill: parent
            Layout.alignment: Qt.AlignHCenter | Qt.AlignVCenter
            spacing: 6

            Item {
                Layout.alignment: Qt.AlignRight | Qt.AlignVCenter
                height: parent.height
                width: 100
                
                MenuItem {
                    text: "INICIAR"
                    active: (activeMenuItem == "launch")

                    onClicked: function() {
                        activeMenuItem = "launch"
                        contentLoader.source = menuSources.launch
                    }
                }
            }

            Item {
                Layout.alignment: Qt.AlignRight | Qt.AlignVCenter
                height: parent.height
                width: 100
                
                MenuItem {
                    width: 100
                    text: "COMUNIDAD"
                    active: (activeMenuItem == "community")

                    onClicked: function() {
                        Qt.openUrlExternally("https://reddit.com/r/manaosdiablo/")
                    }
                }
            }

            Item {
                Layout.alignment: Qt.AlignRight | Qt.AlignVCenter
                height: parent.height
                width: 100
                
                MenuItem {
                    text: "FACEBOOK"
                    active: (activeMenuItem == "ladder")

                    onClicked: function() {
                        Qt.openUrlExternally("https://www.facebook.com/ManaosDiablo")
                    }
                }
            }

            Item {
                Layout.alignment: Qt.AlignRight | Qt.AlignVCenter
                height: parent.height
                width: 100
                
                MenuItem {
                    text: "DISCORD"
                    active: (activeMenuItem == "ladder")

                    onClicked: function() {
                        Qt.openUrlExternally("https://discord.com/invite/QVNnzDR")
                    }
                }
            }

            Item {
                Layout.alignment: Qt.AlignRight | Qt.AlignVCenter
                height: parent.height
                width: 100
                
                MenuItem {
                    text: "WHATSAPP"
                    active: (activeMenuItem == "ladder")

                    onClicked: function() {
                        Qt.openUrlExternally("https://chat.whatsapp.com/BxzRkTOQQNA156Ft80tF9u")
                    }
                }
                
            }
        }
    }

    // Settings.
    Item {
        width: 120; height: parent.height
        anchors.right: parent.right
        anchors.rightMargin: 20

        Item {
            width: 120
            height: 20
            anchors.verticalCenter: parent.verticalCenter
            anchors.horizontalCenter: parent.horizontalCenter

            Image {
                id: optionsIcon
                fillMode: Image.Pad
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                width: 16
                height: 16
                source: "assets/icons/settings.png"
                opacity: settingsHovered ? 1.0 : 0.5

                MouseArea {
                    anchors.fill: parent
                    hoverEnabled: true
                    cursorShape: Qt.PointingHandCursor
                    onClicked: settingsPopup.open()
                    onEntered: {
                        settingsHovered = true
                    }
                    onExited: {
                        settingsHovered = false
                    }
                }
            }
        }
    }

    Separator{
        color: "#161616"
    }
}
