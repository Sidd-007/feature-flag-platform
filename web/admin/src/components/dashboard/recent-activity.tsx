"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { formatRelativeTime } from "@/lib/utils";

interface ActivityItem {
    id: string;
    type: "flag_created" | "flag_updated" | "experiment_started" | "experiment_completed" | "user_added";
    title: string;
    description: string;
    user: {
        name: string;
        email: string;
        avatar?: string;
    };
    timestamp: Date;
}

const activities: ActivityItem[] = [
    {
        id: "1",
        type: "flag_created",
        title: "New feature flag created",
        description: "Created 'new-checkout-flow' flag",
        user: {
            name: "Sarah Chen",
            email: "sarah@example.com",
            avatar: "/avatars/sarah.jpg",
        },
        timestamp: new Date(Date.now() - 1000 * 60 * 5), // 5 minutes ago
    },
    {
        id: "2",
        type: "experiment_started",
        title: "Experiment launched",
        description: "Started A/B test for homepage redesign",
        user: {
            name: "Mike Johnson",
            email: "mike@example.com",
            avatar: "/avatars/mike.jpg",
        },
        timestamp: new Date(Date.now() - 1000 * 60 * 30), // 30 minutes ago
    },
    {
        id: "3",
        type: "flag_updated",
        title: "Flag configuration updated",
        description: "Modified targeting rules for 'premium-features'",
        user: {
            name: "Emma Wilson",
            email: "emma@example.com",
            avatar: "/avatars/emma.jpg",
        },
        timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2), // 2 hours ago
    },
    {
        id: "4",
        type: "experiment_completed",
        title: "Experiment completed",
        description: "Button color test reached significance",
        user: {
            name: "Alex Rodriguez",
            email: "alex@example.com",
            avatar: "/avatars/alex.jpg",
        },
        timestamp: new Date(Date.now() - 1000 * 60 * 60 * 4), // 4 hours ago
    },
    {
        id: "5",
        type: "user_added",
        title: "New team member added",
        description: "Added David Kim to Development team",
        user: {
            name: "Sarah Chen",
            email: "sarah@example.com",
            avatar: "/avatars/sarah.jpg",
        },
        timestamp: new Date(Date.now() - 1000 * 60 * 60 * 24), // 1 day ago
    },
];

function getActivityIcon(type: ActivityItem["type"]) {
    switch (type) {
        case "flag_created":
            return "🚩";
        case "flag_updated":
            return "🔧";
        case "experiment_started":
            return "🧪";
        case "experiment_completed":
            return "✅";
        case "user_added":
            return "👤";
        default:
            return "📝";
    }
}

function getActivityColor(type: ActivityItem["type"]) {
    switch (type) {
        case "flag_created":
            return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
        case "flag_updated":
            return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300";
        case "experiment_started":
            return "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300";
        case "experiment_completed":
            return "bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-300";
        case "user_added":
            return "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300";
        default:
            return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300";
    }
}

export function RecentActivity() {
    return (
        <div className="space-y-4">
            {activities.map((activity) => (
                <div key={activity.id} className="flex items-start space-x-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-full text-xs ${getActivityColor(activity.type)}`}>
                        {getActivityIcon(activity.type)}
                    </div>
                    <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between">
                            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                                {activity.title}
                            </p>
                            <p className="text-xs text-muted-foreground">
                                {formatRelativeTime(activity.timestamp)}
                            </p>
                        </div>
                        <p className="text-sm text-muted-foreground">
                            {activity.description}
                        </p>
                        <div className="mt-1 flex items-center space-x-2">
                            <Avatar className="h-4 w-4">
                                <AvatarImage src={activity.user.avatar} alt={activity.user.name} />
                                <AvatarFallback className="text-xs">
                                    {activity.user.name.split(" ").map(n => n[0]).join("")}
                                </AvatarFallback>
                            </Avatar>
                            <span className="text-xs text-muted-foreground">
                                {activity.user.name}
                            </span>
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
}
