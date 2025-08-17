"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
    Activity,
    AlertTriangle,
    BarChart3,
    Flag,
    TrendingUp,
    Users
} from "lucide-react";

interface StatCardProps {
    title: string;
    value: string;
    change: string;
    changeType: "positive" | "negative" | "neutral";
    icon: React.ReactNode;
}

function StatCard({ title, value, change, changeType, icon }: StatCardProps) {
    const changeColor = {
        positive: "text-green-600 dark:text-green-400",
        negative: "text-red-600 dark:text-red-400",
        neutral: "text-muted-foreground",
    }[changeType];

    return (
        <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{title}</CardTitle>
                <div className="text-muted-foreground">{icon}</div>
            </CardHeader>
            <CardContent>
                <div className="text-2xl font-bold">{value}</div>
                <p className={`text-xs ${changeColor}`}>
                    {change} from last month
                </p>
            </CardContent>
        </Card>
    );
}

export function DashboardStats() {
    const stats = [
        {
            title: "Total Flags",
            value: "247",
            change: "+12.5%",
            changeType: "positive" as const,
            icon: <Flag className="h-4 w-4" />,
        },
        {
            title: "Active Experiments",
            value: "18",
            change: "+3",
            changeType: "positive" as const,
            icon: <BarChart3 className="h-4 w-4" />,
        },
        {
            title: "Monthly Users",
            value: "45.2K",
            change: "+20.1%",
            changeType: "positive" as const,
            icon: <Users className="h-4 w-4" />,
        },
        {
            title: "Conversion Rate",
            value: "12.5%",
            changeType: "positive" as const,
            change: "+2.3%",
            icon: <TrendingUp className="h-4 w-4" />,
        },
        {
            title: "API Requests",
            value: "2.4M",
            change: "+15.2%",
            changeType: "positive" as const,
            icon: <Activity className="h-4 w-4" />,
        },
        {
            title: "Issues",
            value: "3",
            change: "-2",
            changeType: "positive" as const,
            icon: <AlertTriangle className="h-4 w-4" />,
        },
    ];

    return (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
            {stats.map((stat) => (
                <StatCard key={stat.title} {...stat} />
            ))}
        </div>
    );
}
