import { Providers } from "@/components/providers/providers";
import { Toaster } from "@/components/ui/toaster";
import type { Metadata } from "next";
import { DM_Sans } from "next/font/google";
import "./globals.css";

const dmSans = DM_Sans({
    subsets: ["latin"],
    weight: ["400", "500", "700"],
    variable: "--font-dm-sans",
});

export const metadata: Metadata = {
    title: {
        default: "Feature Flag Platform",
        template: "%s | Feature Flag Platform",
    },
    description: "Enterprise-grade feature flag and experimentation platform",
    keywords: [
        "feature flags",
        "feature toggles",
        "experimentation",
        "A/B testing",
        "analytics",
        "dashboard",
    ],
    authors: [
        {
            name: "Feature Flag Platform Team",
        },
    ],
    creator: "Feature Flag Platform",
    metadataBase: new URL("https://featureflags.example.com"),
    openGraph: {
        type: "website",
        locale: "en_US",
        url: "https://featureflags.example.com",
        title: "Feature Flag Platform",
        description: "Enterprise-grade feature flag and experimentation platform",
        siteName: "Feature Flag Platform",
    },
    twitter: {
        card: "summary_large_image",
        title: "Feature Flag Platform",
        description: "Enterprise-grade feature flag and experimentation platform",
        creator: "@featureflags",
    },
    icons: {
        icon: "/favicon.ico",
        shortcut: "/favicon-16x16.png",
        apple: "/apple-touch-icon.png",
    },
    manifest: "/site.webmanifest",
};

interface RootLayoutProps {
    children: React.ReactNode;
}

export default function RootLayout({ children }: RootLayoutProps) {
    return (
        <html lang="en" className="dark" suppressHydrationWarning>
            <head />
            <body className={`${dmSans.className} min-h-screen bg-background text-foreground antialiased`}>
                <Providers>
                    {children}
                    <Toaster />
                </Providers>
            </body>
        </html>
    );
}
