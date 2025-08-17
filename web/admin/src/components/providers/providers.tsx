"use client";

import { TooltipProvider } from "@/components/ui/tooltip";
import { DataProvider } from "@/providers/DataProvider";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { ThemeProvider } from "next-themes";
import { useState } from "react";

interface ProvidersProps {
    children: React.ReactNode;
}

export function Providers({ children }: ProvidersProps) {
    const [queryClient] = useState(
        () =>
            new QueryClient({
                defaultOptions: {
                    queries: {
                        staleTime: 60 * 1000, // 1 minute
                        cacheTime: 5 * 60 * 1000, // 5 minutes
                        retry: (failureCount, error: any) => {
                            // Don't retry on 4xx errors
                            if (error?.status >= 400 && error?.status < 500) {
                                return false;
                            }
                            return failureCount < 3;
                        },
                        refetchOnWindowFocus: false,
                    },
                    mutations: {
                        retry: false,
                    },
                },
            })
    );

    return (
        <QueryClientProvider client={queryClient}>
            <ThemeProvider
                attribute="class"
                defaultTheme="dark"
                enableSystem={false}
                forcedTheme="dark"
                disableTransitionOnChange
            >
                <DataProvider>
                    <TooltipProvider>
                        {children}
                    </TooltipProvider>
                </DataProvider>
            </ThemeProvider>
            <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
    );
}
