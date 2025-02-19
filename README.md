## Portable Control Panel for owlcms-firmata Server

[owlcms-firmata]([url](https://github.com/jflamy/owlcms-firmata)) it a software program to control refereeing/jury/timekeeper/announcer devices and interact with an owlcms server.
owlcms-firmata is installed on the computers where the devices are plugged in.

This provides a small compiled binary to launch owlcms-firmata, removing the need for an installer
and providing the same user experience on all platforms. The program allows

- easy updating from a version to the current one
- launching the program (downloading Java if needed)
- having several versions at once and copying configurations and data

Currently supported: Windows, Raspberry Pi, Linux on Intel<br>
Contact us if willing to test on Mac.

![image](https://github.com/user-attachments/assets/245a8b2f-1116-499f-ab43-63e654913559)



### Usage
1. go to the Releases page and download the installer or program for your type of computer
3. If there is no version of firmata installed, the latest one will be downloaded
4. Click Launch to start firmata
   - This will create a folder called java17 the first time
   - Starting the program takes 10 to 20 seconds depending on your laptop, the time it takes to read in and process the various configuration files and read in the database
5. You can hide the window until you need to stop the program.
   - You can either use the Stop button or the stop icon (X or red dot) at the top of the program

### Device Configuration

> Note: the configuration files are located in the `config` subdirectory of the installation
> - Use the `Files` button for the version to reach it.
