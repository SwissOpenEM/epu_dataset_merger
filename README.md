# EPU/Tomo5 to data Mover

Small tool to move some of the data and the metadata that EPU usually keeps on the microscope computer to your data directories. As these two directories are often not on the same computer this will require some mounts to be set.

## What it does
* EPU/ Single particle Analysis: Copies corresponding xmls (metadata output) next to the tiff/eer/mrc movies, optionally also adds the Atlas to the top level directory
* Tomo5: Copies some overview xmls, the Batch folder with its xmls and the entirety of the SearchMaps folder. Optionally also adds the Atlas to the top level directory


## Requirements
* EPU Mountpoint: This is the directory on the microscope computer that mirrors your data output folder in terms of folder nameing 
* Atlas Mountpoint(Optional): Depending on your setup these might be on a different system or not, so it really depends - in the setup this was written for they have their own partition.

## Flags:
* --i point to your EPU folder on the microscope computer
* --o point to your dataset folder (where full datasets are output to) TFS typical: "OffloadData". This one is usually NOT on the microscope computer
* --a Optional, point to atlas folder if you want the overview atlas with your data. Sometimes these can also be in the EPU folder - in that case point it there. The flag is required to be set if you want the atlas.

## Where and how o deploy
Depending on your setup and the mountpoints you can run this on a (linux) machine that has access to both the datasets and the EPU folders (and the atlas one if you want those). Alternatively it is Windows compatible so you can directly run it on the mircoscope computer. We recommend using task scheduler, systemd or similar to automatize it in the background. Running every 15-20 minutes has worked well in our experience (even with a lot of datasets runtime of the merger should be <5 min). 

