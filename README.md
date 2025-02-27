# EPU/Tomo5 to data Mover

Small tool to move some of the data and the metadata that EPU usually keeps on the microscope computer to you data directories. As these two directories are often not on the same computer this will require some mounts to be set.

## What it does
* EPU/ Single particle Analysis: Copies corresponding xmls (metadata output) next to the tiff/eer/mrc movies, optionally also adds the Atlas to the top level directory
* Tomo5: Copies some overview xmls, the Batch folder with its xmls and the entirety of the SearchMaps folder. Optionally also adds the Atlas to the top level directory


## Requirements
* EPU Mountpoint: This is the directory on the microscope computer that mirrors your data output folder in terms of folder nameing 
* Atlas Mountpoint(Optional): Depending on your setup these might be on a different system or not, so it really depends - in the setup this was written for they have their own partition.

## Flags:
* --i wants your EPU folder
* --o should point to your data folder (where full datasets are output to)
* --a Optional, if you want an Atlas and have it on a different directory.

