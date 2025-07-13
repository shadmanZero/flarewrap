package image


type Image struct {
	ImageRef  string
	ImageName string
}
func NewImage(imageRef string, imageName string) *Image {
	return &Image{
		ImageRef:  imageRef,
		ImageName: imageName,
	}
}

