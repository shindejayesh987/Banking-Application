package assignment0.OOAD;

import java.util.Scanner;

public class BankTester 
{
	public static void main(String args[]) throws Exception		//Main Class of the 3 java files
	{
		int choice = 1;
		Bank bank = new Bank();
		
		while(choice!=2)										
		{
			System.out.println("Bank User Options:");			//Give Option to login or save and quit.
			System.out.println("1. Login");
			System.out.println("2. Save & Quit");
			@SuppressWarnings("resource")
			Scanner reader = new Scanner(System.in);
			choice = reader.nextInt();
			
			if (choice == 1)
			{
				bank.login();									//This while loop will continuously prompt the user to 
																//with the two options while the user continues to pick
																//login option until option 2 is selected 
				choice = 0;
			}
			else if(choice == 2)
			{
				System.out.println("Thank You! Quiting now.");
			}
			else
			{
				System.out.println("Choose from given options!!!");
			}
		}
		
		if(choice == 2)
		{
			bank.write_DB();									//This method is called to save the changes made in file
		}
		
	}
}
