import type { Metadata } from "next";
import { AppRouterCacheProvider } from "@mui/material-nextjs/v16-appRouter";
import { AppProviders } from "@/components/app-providers";
import "./globals.css";

export const metadata: Metadata = { title: { default: "Trip Planner — Make a trip of your own", template: "%s · Trip Planner" }, description: "Find places worth stopping for, shape your days, and keep your itinerary close." };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body><AppRouterCacheProvider options={{ enableCssLayer: true }}><AppProviders>{children}</AppProviders></AppRouterCacheProvider></body></html>;
}
