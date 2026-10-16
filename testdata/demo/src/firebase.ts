import { initializeApp } from "firebase/app";
import { getAuth } from "firebase/auth";

export const app = initializeApp({
  apiKey: "demo",
  authDomain: "shop-demo.firebaseapp.com",
  projectId: "shop-demo",
});
export const auth = getAuth(app);
