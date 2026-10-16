import { GoogleMap, useJsApiLoader } from "@react-google-maps/api";

export function StoreMap() {
  const { isLoaded } = useJsApiLoader({ googleMapsApiKey: "demo" });
  return isLoaded ? <GoogleMap zoom={12} /> : null;
}
