//找到最短路径
#include<stdio.h>
#include<string.h>
int n,s,d;
int road[100][100];
int way[100][100];
int minnum;
int short_path[100];
int visited[100];
int current_path[100];
int short_path_len;
int current_path_len;//城市数量
void dfs(int s,int d,int add)
{
	visited[s]=1;
	current_path[current_path_len++]=s;
	if(s==d){
		if(add<minnum){
			minnum=add;
			short_path_len=current_path_len;
			for(int i=0;i<current_path_len;i++){
				short_path[i]=current_path[i];
			}
		}
	
	}
	for(int j=0;j<n;j++){
		if(road[s][j]!=-1&&visited[j]!=1){
	
			
			dfs(j,d,add+road[s][j]);
			
			
		}
	}
	visited[s]=0;
	current_path_len--;	
}
int main()
{
	scanf("%d",&n);
	scanf("%d %d",&s,&d);
	
	for(int i=0;i<n;i++){
		for(int j=0;j<n;j++){
			scanf("%d",&road[i][j]);
		}
	}
	
	minnum=10000;
	short_path_len=0;
	current_path_len=0;
	for(int i=0;i<n;i++){
		visited[i]=0;
	}
	dfs(s,d,0);
	if(minnum!=10000){
		for(int i=0;i<short_path_len;i++){
			if(i<short_path_len-1){
				printf("%d->",short_path[i]);
			}else{
				printf("%d\n",short_path[i]);
				}
			}
			return 0;
	}else{
		printf("-1");
	}
	return 0;	
}
