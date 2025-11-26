# Pyramid Schema


Important links:

https://github.com/libvips/libvips
https://www.libvips.org/API/current/making-image-pyramids.html


#### Key terms & Useful CLI tools/info
- libvips, vipsgen (Go binding of libvips)
  - vipsheader (What libvips can see)
- tiffinfo -> a really useful cli tool that just outputs a lot of tiff metadata 

Tiffs can have multiple pages ("tiff directories") within the same file. These are often
different 'images' or element maps of the same image, but can also be completely different.


- Subifds, also known as "levels" of the pyramids, and these sometimes exist within TIFF pages.
    Some pages may have pyramids while others do not. This is not relevant for us - we generate with DzSave. In the end, everything uses libtiff.




### Standards
- We use DeepZoom (some microsoft standard viewing/tile structure layout).
- 
### Pipeline
We initially had the importer directly Tiffload the .tif and DzSave the whole pyramid jpgs to temp, then upload that to S3.
However, this has a slight issue: If the pyramid exceeds the size /tmp can store, well, that's not good! Jus completely breaks.

So our current solution is now to do this pipeline for each page:
- Open the original tiff file w/ NewTiffLoad (doesn't load the whole thing in memory)
- Read the current page 
- DzSave that to get some pyramid output in /tmp
- Upload that to S3
- Delete that directory locally in the tmp
- Repeat


### Extra things

- Libvips Tiffload throws an error if you try to read multiple pages and ANY of those pages have differing dimensions.