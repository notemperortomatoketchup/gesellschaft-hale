"use client";
import { QueryClient, QueryClientProvider } from "react-query";
import "./globals.css";
import { Inter } from "next/font/google";

const inter = Inter({ subsets: ["latin"] });

export default function RootLayout({ children }) {
  const queryClient = new QueryClient();

  return (
    <html lang="en">
      <QueryClientProvider client={queryClient}>
        <body className={`${inter.className} h-full w-full dark`}>
          {children}
        </body>
      </QueryClientProvider>
    </html>
  );
}
