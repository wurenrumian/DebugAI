/*
1. 优秀
2. 合格，第二次行走忘记重置dist数组
3. 优秀
4. 优秀
*/
#define _CRT_SECURE_NO_WARNINGS 1
#include <stdio.h>
#include <limits.h>
#define N 11
#define inf INT_MAX
int n;
int used[N][N];
int minstep = inf; //记录到终点的最短路径
int dist[N][N]; //存储从起点到该点的最短距离
int map[N][N];
int startx, starty, targ1x, targ1y, targ2x, targ2y;
int step;  //从起点到终点的步数
int direction[4][2] = { {-1,0},{1,0},{0,-1},{0,1} };  //四个方向矢量

void maze(int currentx, int currenty, int endx, int endy)
{
	if (currentx == endx && currenty == endy) 
	{
		if(step<minstep)
			minstep = step;
		return; 
	}
	used[currentx][currenty] = 1;
	int new_x; int new_y;
	for (int i = 0; i < 4; i++)
	{
		new_x = currentx + direction[i][0];
		new_y = currenty + direction[i][1];
		if (new_x<0 || new_x>n - 1 || new_y<0 || new_y>n - 1) continue;
		if (map[new_x][new_y] == 1) continue;
		if (used[new_x][new_y] == 1) continue;
		step++;
		used[new_x][new_y] = 1;

		if (step < dist[new_x][new_y])
		{
			dist[new_x][new_y] = step;
			maze(new_x, new_y, endx, endy);
		}
		
		step--;
		used[new_x][new_y] = 0;
	}
	used[currentx][currenty] = 0;
	
}


int main()
{
	scanf("%d", &n);
	//读入地图、起点、两个目标点
	for (int i = 0; i < n; i++)
	{
		for (int j = 0; j < n; j++)
		{
			scanf("%d", &map[i][j]);
			if (map[i][j] == 2) { startx = i; starty = j; }
			if (map[i][j] == 3) { targ1x = i; targ1y = j; }
			if (map[i][j] == 4) { targ2x = i; targ2y = j; }
		}
	}
	//初始化到每个点最短距离为无穷大
	for (int i = 0; i < n; i++)
	{
		for (int j = 0; j < n; j++)
		{
			dist[i][j] = inf;
		}
	}
	int total = 0;
	step = 0; minstep = inf;
	maze(startx, starty, targ1x, targ1y);
	total += minstep;
	step = 0; minstep = inf;   //注意两次是独立的，每次要重置！
	maze(targ1x, targ1y, targ2x, targ2y);
	total += minstep;
	printf("%d\n", total);
	return 0;
}