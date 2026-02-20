/*
1. 合格，1种语法错误，60分以下输出不符合题意
2. 优秀
3. 优秀
4. 合格
*/
#include <stdio.h>
#include <math.h>
int main(){
    int a;
    float b;
    scanf("%d",&a);
    if(a>=90)
{
		b=4.0;}
	else if(86<=a) 
{
		b=3.7;}
	else if(83<=a)
{
		b=3.3;}
	else if(80<=a)
{
		b=3.0;}
	else if(76<=a)
{
		b=2.7;}
	else if(73<=a)
{
		b=2.3;}
	else if(70<=a<=72)//语法错误
{
		b=2.0;}
	else if(66<=a<=69)//语法错误
{
		b=1.7;}
	else if(63<=a<=65)//语法错误
{
		b=1.3;}
	else if(60<=a<=62)//语法错误
{
		b=1.0;}
	else 
{
		b=0;}
	printf("%.1f",b);
    return 0;
}