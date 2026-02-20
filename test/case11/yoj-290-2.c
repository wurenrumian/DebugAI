#include<iostream>
#include<cstdio>
#include<algorithm>
#define Maxx 150005
int person_num,shop_num,t,cnt;
int things[Maxx];
bool flag;
int cnts[Maxx];
int main(){
	scanf("%d",&person_num);
	for(int i=1;i<=person_num;i++){
		scanf("%d",&shop_num); 
		for(int j=1;j<=shop_num;j++){
			scanf("%d",&t);
			if(i==1)
			things[cnt]=t,cnts[cnt]++,cnt++;
			else{
				for(int i=1;i<cnt;i++)
				if(things[i]==t)
				cnts[i]++;
			}
		}
		if(i==1)
		std::sort(things+1,things+cnt);
	}
	for(int i=1;i<cnt;i++)
	if(cnts[i]==person_num)
	printf("%d ",things[i]),flag=1;
	if(!flag)
	printf("NO");
	return 0;
}